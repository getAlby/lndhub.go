package lib

import (
	"github.com/labstack/echo/v4"
	"github.com/uptrace/bun"
)

type LndhubContext struct {
	echo.Context

	DB *bun.DB
}
