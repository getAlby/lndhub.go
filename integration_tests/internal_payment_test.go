package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type InternalPaymentTestSuite struct {
	suite.Suite
}

func (InternalPaymentTestSuite) SetupSuite() {

}

func (InternalPaymentTestSuite) TearDownSuite() {

}

func (InternalPaymentTestSuite) TestAddInvoice() {

}

func TestInternalPaymentTestSuite(t *testing.T) {
	suite.Run(t, new(InternalPaymentTestSuite))
}
