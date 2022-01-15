package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bumi/lndhub.go/pkg/controllers"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestCreateUser(t *testing.T) {
	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	c.SetParamNames("login", "password")
	c.SetParamValues("test-login", "test-password")

	if assert.NoError(t, controllers.CreateUserController{}.CreateUser(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}
