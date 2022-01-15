package middlewares

import (
	"github.com/labstack/echo/v4"
	"github.com/uptrace/bun"
)

// ContextDB : pass db
func ContextDB(db *bun.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("db", db)
			return next(c)
		}
	}
}
