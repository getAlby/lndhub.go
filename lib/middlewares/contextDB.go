package middlewares

import (
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
)

// ContextDB : pass db
func ContextDB(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("db", db)
			return next(c)
		}
	}
}
