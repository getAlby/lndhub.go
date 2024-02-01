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


func (svc *LndhubService) ValidateNostrEventPayload() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			// Validate Payload
			// TODO | CLEANUP - consolidate this code with the EventBody struct in nostr.ctrl.go
			type Payload struct {
				ID        string            `json:"ID"`
				Pubkey    string            `json:"Pubkey"`
				CreatedAt int64             `json:"CreatedAt"`
				Kind      int               `json:"kind"`
				Tags      [][]interface{}   `json:"tags"`
				Content   string            `json:"Content"`
				Sig       string            `json:"Sig"`
				Addr      string            `json:"Addr"`
				Fee       float64           `json:"Fee"`
			}

			var payload Payload

			switch payload.Content {
			// TODO | CLEANUP -  move these constants to common/globals.go
			case "TAHUB_CREATE_USER":

				if payload.Kind != 1 {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'kind' must be 1",
					})
				}
				return next(c)

			case  "TAHUB_GET_BALANCES":

				if payload.Kind != 1 {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'kind' must be 1",
					})
				}
				return next(c)

			case "TAHUB_RECEIVE_ADDRESS_FOR_ASSET":
				// Validate specific fields for TAHUB_RECEIVE_ADDRESS_FOR_ASSET event
				if payload.Kind != 1 {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'kind' must be 1",
					})
				}
					
				if len(payload.Tags) == 0 {
						return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
							"error":   true,
							"code":    2,
							"message": "Field 'tags' must exist and not be empty",
						})
					}

					// Check 'Ta' and 'Amt' in the 'tags' array
					var taExists, amtExists bool
					for _, tag := range payload.Tags {
						if len(tag) == 2 {
							key, ok := tag[0].(string)
							if !ok {
								continue
							}
							value, ok := tag[1].(string)
							if !ok {
								continue
							}
							if key == "ta" && value != "" {
								taExists = true
							} else if key == "amt" && value != "" {
								amtExists = true
							}
						}
					}

					if !taExists || !amtExists {
						return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
							"error":   true,
							"code":    2,
							"message": "Fields 'ta' and 'amt' must exist in 'tags' array with values",
						})
					}

					return next(c)

			case "TAHUB_SEND_ASSET":
				// Validate specific fields for TAHUB_SEND_ASSET event
				if payload.Kind != 1 {
					return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
						"error":   true,
						"code":    2,
						"message": "Field 'kind' must be 1",
					})
				}
					
					if len(payload.Tags) == 0 {
						return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
							"error":   true,
							"code":    2,
							"message": "Field 'tags' must exist and not be empty",
						})
					}
			
					// Check 'addr' and 'fee' in the 'tags' array
					var addrExists, feeExists bool
					for _, tag := range payload.Tags {
						if len(tag) == 2 {
							key, ok := tag[0].(string)
							if !ok {
								continue
							}
							switch key {
							case "addr":
								if value, ok := tag[1].(string); ok && value != "" {
									addrExists = true
								}
							case "fee":
								if value, ok := tag[1].(float64); ok && value != 0 {
									feeExists = true
								}
							}
						}
					}
			
					if !addrExists || !feeExists {
						return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
							"error":   true,
							"code":    2,
							"message": "Fields 'addr' and 'fee' must exist in 'tags' array and not be empty",
						})
					}
			
					return next(c)
				
			
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

func (svc *LndhubService) ValidateUserMiddleware() echo.MiddlewareFunc {
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
