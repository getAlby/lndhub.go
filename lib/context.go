package lib

import (
	"github.com/bumi/lndhub.go/db/models"
	"github.com/labstack/echo/v4"
	"github.com/uptrace/bun"
)

type LndhubContext struct {
	echo.Context

	DB   *bun.DB
	User *models.User
}
