package create

import (
	"github.com/labstack/gommon/random"
	"gorm.io/gorm"
	"math/rand"
	"net/http"

	"github.com/bumi/lndhub.go/database/models"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

const letterBytes = random.Alphanumeric

func (CreateUserRouter) CreateUser(c echo.Context) error {
	type RequestBody struct {
		PartnerID   string `json:"partnerid"`
		AccountType string `json:"accounttype"`
	}
	var body RequestBody

	body.PartnerID = c.FormValue("partnerid")
	body.AccountType = c.FormValue("accounttype")

	db, _ := c.Get("db").(*gorm.DB)

	user := &models.User{}

	user.Login = RandStringBytes(8)
	user.Password = RandStringBytes(15)

	result := db.Create(&user)

	logrus.Printf("%v", result)
	return c.JSON(http.StatusOK, echo.Map{
		"user": user,
	})
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
