package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bumi/lndhub.go/pkg/controllers"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestAuth(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/auth", nil)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	c.SetParamNames("access_token", "test-access-token")
	c.SetParamValues("refresh_token", "test-refresh-token")

	if assert.NoError(t, controllers.AuthController{}.Auth(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
