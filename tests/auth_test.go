package tests

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type UserAuthTestSuite struct {
	suite.Suite
}

func (UserAuthTestSuite) SetupSuite() {

}

func (UserAuthTestSuite) TearDownSuite() {

}

func (UserAuthTestSuite) TestAuth() {

}

func TestUserAuthTestSuite(t *testing.T) {
	suite.Run(t, new(UserAuthTestSuite))
}
