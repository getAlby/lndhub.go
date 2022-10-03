package tokens

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func AdminTokenMiddleware(token string) echo.MiddlewareFunc {
	if token == "" {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}
	return middleware.KeyAuth(func(auth string, c echo.Context) (bool, error) {
		return auth == token, nil
	})
}
