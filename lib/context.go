package lib

import (
	"github.com/labstack/echo/v4"
	"github.com/uptrace/bun"
)

type IndhubContext struct {
	echo.Context

	DB *bun.DB
}
