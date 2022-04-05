package responses

import (
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestBadAuthErrorsNotAllowedForSentry(t *testing.T) {
	badAuthErrResponse := echo.NewHTTPError(http.StatusBadRequest, echo.Map{
		"error":   true,
		"code":    1,
		"message": "bad auth",
	})

	isAllowed := isErrAllowedForSentry(badAuthErrResponse)
	assert.False(t, isAllowed)
}

func TestNotBadAuthErrorsAllowedForSentry(t *testing.T) {
	notBadAuthErrResponse := echo.NewHTTPError(http.StatusBadRequest, echo.Map{
		"error":   true,
		"code":    2,
		"message": "not bad auth",
	})

	isAllowed := isErrAllowedForSentry(notBadAuthErrResponse)
	assert.True(t, isAllowed)
}

func TestNonErrorResponseErrorsAllowedForSentry(t *testing.T) {
	err := errors.New("random error")

	isAllowed := isErrAllowedForSentry(err)
	assert.True(t, isAllowed)
}
