package create

import (
	"github.com/bumi/lndhub.go/lib/security"
	"gorm.io/gorm"
	"math/rand"
	"net/http"

	"github.com/bumi/lndhub.go/database/models"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
)

const alphaNumBytes = random.Alphanumeric

// CreateUserRouter : Create user router struct
type CreateUserRouter struct{}

// CreateUser : Create user Router
func (CreateUserRouter) CreateUser(c echo.Context) error {
	type RequestBody struct {
		PartnerID   string `json:"partnerid"`
		AccountType string `json:"accounttype"`
	}
	var body RequestBody

	if err := c.Bind(&body); err != nil {
		return err
	}

	db, _ := c.Get("db").(*gorm.DB)

	user := &models.User{}

	user.Login = RandStringBytes(8)
	user.Password = RandStringBytes(15)
	security.HashPassword(&user.Password)
	db.Create(&user)

	var ResponseBody struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	ResponseBody.Login = user.Login
	ResponseBody.Password = user.Password

	return c.JSON(http.StatusOK, &ResponseBody)
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphaNumBytes[rand.Intn(len(alphaNumBytes))]
	}
	return string(b)
}
