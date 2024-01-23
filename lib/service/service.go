package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/getAlby/lndhub.go/rabbitmq"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/tokens"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
	"github.com/uptrace/bun"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/crypto/bcrypt"
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

func (svc *LndhubService) GenerateToken(ctx context.Context, login, password, inRefreshToken string) (accessToken, refreshToken string, err error) {
	var user models.User

	switch {
	case login != "" || password != "":
		{
			if err := svc.DB.NewSelect().Model(&user).Where("login = ?", login).Scan(ctx); err != nil {
				return "", "", fmt.Errorf("bad auth")
			}
			if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
				return "", "", fmt.Errorf("bad auth")
			}
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

	if user.Deactivated {
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


func (svc *LndhubService) ValidateEventPayload () echo.MiddlewareFunc {
	return func (next echo.HandlerFunc) echo.HandlerFunc {
		return func (c echo.Context) error {

			// Validate Payload
			var payload struct {
				ID        string `json:"ID"`
				Pubkey    string `json:"Pubkey"`
				CreatedAt string `json:"CreatedAt"`
				Kind      int    `json:"kind"`
				Ta        string `json:"Ta"`
				Amt       float64   `json:"Amt"`
				Addr      string   `json:"addr"`
				Fee       float64  `json:"fee"`
				Content   string `json:"Content"`
				Sig       string `json:"Sig"`
			}

			if err := c.Bind(&payload); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
					"error":   true,
					"code":    2,
					"message": "Bad request payload",
				})
			}

			switch payload.Content {
			case "TAHUB_CREATE_USER", "TAHUB_GET_BALANCES":
			
				if payload.Kind != 1 {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'Kind' must be 1",
					})
					
				}
				return next((c))
			case "TAHUB_RECEIVE_ADDRESS_FOR_ASSET":
				// Validate specific fields for TAHUB_RECEIVE_ADDRESS_FOR_ASSET event
				if payload.Kind != 1 {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'Kind' must be 1",
					})
				}
		
				if len(payload.Ta) == 0 || payload.Ta == "" {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'ta' must exist and not be empty",
					})
					
				}
				if payload.Amt < 0 || payload.Amt != float64(int64(payload.Amt)) {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'amt' must be a positive integer (u64)",
					})
					
				}
				return next((c))
			case "TAHUB_SEND_ASSET":
				// Validate specific fields for TAHUB_SEND_ASSET event
				if payload.Kind != 1 {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'Kind' must be 1",
					})
				}
		
				if len(payload.Addr) == 0 || payload.Addr == "" {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'addr' must exist and not be empty",
					})
				// 	return errors.New("Field 'addr' must exist and not be empty")
				}
				if payload.Fee < 0 || payload.Fee != float64(int64(payload.Fee)) {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'fee' must be a positive integer (u64)",
					})
					// return errors.New("Field 'fee' must be a positive integer (u64)")
				}
				return next((c))
			default:
				return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
					"error":   true,
					"code":    2,
					"message": "Invalid event content",
				})
				
			}

		}
	}
}