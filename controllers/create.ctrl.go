package controllers

import (
	"context"
	"database/sql"
	"math/rand"
	"net/http"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lib/security"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
	"github.com/uptrace/bun"
)

const alphaNumBytes = random.Alphanumeric

// CreateUserController : Create user controller struct
type CreateUserController struct{}

// CreateUser : Create user Controller
func (CreateUserController) CreateUser(c echo.Context) error {
	ctx := c.(*lib.LndhubContext)

	// optional parameters that we currently do not use
	type RequestBody struct {
		PartnerID   string `json:"partnerid"`
		AccountType string `json:"accounttype"`
	}
	var body RequestBody

	if err := c.Bind(&body); err != nil {
		return err
	}

	db := ctx.DB

	user := models.User{}

	// generate user login/password (TODO: allow the user to choose a login/password?)
	user.Login = randStringBytes(8)
	password := randStringBytes(15)
	// we only store the hashed password but return the initial plain text password in the HTTP response
	hashedPassword := security.HashPassword(password)
	user.Password = hashedPassword

	// Create user and the user's accounts
	// We use double-entry bookkeeping so we use 4 accounts: incoming, current, outgoing and fees
	// Wrapping this in a transaction in case something fails
	err := db.RunInTx(context.TODO(), &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(&user).Exec(ctx); err != nil {
			return err
		}
		accountTypes := []string{"incoming", "current", "outgoing", "fees"}
		for _, accountType := range accountTypes {
			account := models.Account{UserID: user.ID, Type: accountType}
			if _, err := db.NewInsert().Model(&account).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})

	// Was the DB transaction successful?
	if err != nil {
		return err
	}

	var ResponseBody struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	ResponseBody.Login = user.Login
	ResponseBody.Password = password

	return c.JSON(http.StatusOK, &ResponseBody)
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphaNumBytes[rand.Intn(len(alphaNumBytes))]
	}
	return string(b)
}
