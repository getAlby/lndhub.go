package tests

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type CreateUserTestSuite struct {
	suite.Suite
}

func (CreateUserTestSuite) SetupSuite() {

}

func (CreateUserTestSuite) TearDownSuite() {

}

func (CreateUserTestSuite) TestCreate() {

}

func TestCreateUserTestSuite(t *testing.T) {
	suite.Run(t, new(CreateUserTestSuite))
}
