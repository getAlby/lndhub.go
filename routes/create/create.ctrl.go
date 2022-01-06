package create

import (
	"github.com/bumi/lndhub.go/database/models"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"net/http"
)

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
	//ToDo random login func
	user.Login = "random"
	//ToDo random password func
	user.Password = "random"


	result := db.Create(&user)

	logrus.Printf("%v", result)
	return c.JSON(http.StatusOK, echo.Map{
		"user":  user,
	})
}
