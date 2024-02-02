package service

import (
	"context"
	//"crypto/rand"
	"errors"
	"fmt"
	//"math/big"
	"net/http"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/getAlby/lndhub.go/rabbitmq"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
	"github.com/nbd-wtf/go-nostr"
	"github.com/uptrace/bun"
	"github.com/ziflex/lecho/v3"
	//"golang.org/x/crypto/bcrypt"
)

const alphaNumBytes = random.Alphanumeric

type LndhubService struct {
	Config         *Config
	DB             *bun.DB
	LndClient      lnd.LightningClientWrapper
	RabbitMQClient rabbitmq.Client
	Logger         *lecho.Logger
	InvoicePubSub  *Pubsub
}

// type EventRequestBody struct {
// 	ID        string            `json:"id"`
// 	Pubkey    string            `json:"pubkey"`
// 	CreatedAt int64             `json:"created_at"`
// 	Kind      int64               `json:"kind"`
// 	Tags      [][]interface{}   `json:"tags"`
// 	Content   string            `json:"content"`
// 	Sig       string            `json:"sig"`
// }

func (svc *LndhubService) GenerateToken(ctx context.Context, login, password, inRefreshToken string) (accessToken, refreshToken string, err error) {
	var user models.User

	switch {
	// TODO adjust this function to authenticate user with the previously registered pubkey
	//		and the signature on the current event - when required to do so
	case login != "" || password != "":
		{
			if err := svc.DB.NewSelect().Model(&user).Where("login = ?", login).Scan(ctx); err != nil {
				return "", "", fmt.Errorf("bad auth")
			}
			// if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
			// 	return "", "", fmt.Errorf("bad auth")
			// }
		}
	case inRefreshToken != "":
		{
			userId, err := tokens.GetUserIdFromToken(svc.Config.JWTSecret, inRefreshToken)
			if err != nil {
				return "", "", fmt.Errorf("bad auth")
			}

			if err := svc.DB.NewSelect().Model(&user).Where("id = ?", userId).Scan(ctx); err != nil {
				return "", "", fmt.Errorf("bad auth")
			}
		}
	default:
		{
			return "", "", fmt.Errorf("login and password or refresh token is required")
		}
	}

	if user.Deactivated || user.Deleted {
		return "", "", fmt.Errorf(responses.AccountDeactivatedError.Message)
	}

	accessToken, err = tokens.GenerateAccessToken(svc.Config.JWTSecret, svc.Config.JWTAccessTokenExpiry, &user)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = tokens.GenerateRefreshToken(svc.Config.JWTSecret, svc.Config.JWTRefreshTokenExpiry, &user)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (svc *LndhubService) ParseInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case string:
		c, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, err
		}
		return c, nil
	default:
		return 0, fmt.Errorf("conversion to int from %T not supported", v)
	}
}

func (svc *LndhubService) VerfiySchnorrSig(event nostr.Event) {
	// decode public key
	
}

func (svc *LndhubService) CheckEvent(payload nostr.Event) (bool, error) {
	
	if payload.Kind != 1 {
		return false, errors.New("Field 'kind' must be 1")
	}
	// TODO perform checks on content
	// check the length of the content 
	if len(payload.Content) == 0 {
		return false, errors.New("Field 'Content' must have a value")
	}
	// Split event content
	data := strings.Split(payload.Content, ":")
	if len(data) == 0 {
		return false, errors.New("Field 'Content' must at least specify the action.")
	}

	switch data[0] {

	case "TAHUB_CREATE_USER":

		return true, nil
		
	case "TAHUB_RECEIVE_ADDRESS_FOR_ASSET":
		// this action must have three parts to the content
		if len(data) != 3 {
			return false, errors.New("Invalid 'Content' for TAHUB_RECEIVE_ADDRESS_FOR_ASSET.")
		}
		// Validate specific fields for TAHUB_RECEIVE_ADDRESS_FOR_ASSET event

		// TODO come up with further validations for this asset_id i.e. a Taproot Asset AssetID or 'btc'
		// validate asset ID
		if data[1] == "" {
			return false, errors.New("Field 'Asset ID' must have a value")
		}
		// validate amt
		amt, err := strconv.ParseFloat(data[2], 64)
		if err != nil || amt > 0 {
			return false, errors.New("Field 'amt' must be a valid number and non-zero")
		}

		return true, nil

	case "TAHUB_SEND_ASSET":
		// this action must have three parts to the content
		if len(data) != 3 {
			return false, errors.New("Invalid 'Content' for TAHUB_SEND_ASSET.")
		}
		// Validate specific fields for TAHUB_SEND_ASSET event
		// TODO consider other validation on the address
		if data[1] == "" {
				return false, errors.New("Field 'ADDR' must have a value")
		}
		// decode the address (str, bytes, err)
		_, _, err := bech32.Decode(data[1])
		if err != nil {
		return false, err
		}
		// validate amt to send
		amt, err := strconv.ParseFloat(data[2], 64)
		// TODO consider amt thresholds and their implication there
		if err != nil || amt < 0 {
			return false, errors.New("Field 'amt' must be a valid number and non-zero")
		}
		// validate fee for tx
		fee, err := strconv.ParseFloat(data[3], 64)
		// TODO consider fee thresholds, limits, etc. that make sense to validate/apply here
		if err != nil || fee != 0 {
			return false, errors.New("Field 'fee' must be a valid number")
		}

		return true, nil

	case "TAHUB_GET_BALANCES":
		return true, nil  

	default:
		return false, errors.New("Undefined 'Content' Name")
	}
	
}




func (svc *LndhubService) ValidateUserMiddleware() echo.MiddlewareFunc {
	// TODO update ValidateUserMiddlware 
	// * it has already performed a check on the pubkey for the event passed to endpoint
	// * it must know ensure that pubkey returns a user in the database
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userId := c.Get("UserID").(int64)
			if userId == 0 {
				return echo.ErrUnauthorized
			}
			user, err := svc.FindUser(c.Request().Context(), userId)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, echo.Map{
					"error":   true,
					"code":    1,
					"message": "bad auth",
				})
			}
			if user.Deactivated || user.Deleted {
				return echo.NewHTTPError(http.StatusUnauthorized, echo.Map{
					"error":   true,
					"code":    1,
					"message": "bad auth",
				})
			}
			return next(c)
		}
	}
}
