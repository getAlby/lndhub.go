package tests

import (
	"testing"

	"github.com/stretchr/testify/suite"
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
