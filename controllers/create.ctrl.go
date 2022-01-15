package controllers

import (
	"context"
	"math/rand"
	"net/http"

	"github.com/bumi/lndhub.go/db/models"
	"github.com/bumi/lndhub.go/lib"
	"github.com/bumi/lndhub.go/lib/security"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
)

const alphaNumBytes = random.Alphanumeric

// CreateUserController : Create user controller struct
type CreateUserController struct{}

// CreateUser : Create user Controller
func (CreateUserController) CreateUser(c echo.Context) error {
	ctx := c.(lib.IndhubContext)
	type RequestBody struct {
		PartnerID   string `json:"partnerid"`
		AccountType string `json:"accounttype"`
	}
	var body RequestBody

	if err := c.Bind(&body); err != nil {
		return err
	}

	db := ctx.DB

	user := &models.User{}

	user.Login = randStringBytes(8)
	password := randStringBytes(15)
	hashedPassword := security.HashPassword(password)
	user.Password = hashedPassword

	if _, err := db.NewInsert().Model(&user).Exec(context.TODO()); err != nil {
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
