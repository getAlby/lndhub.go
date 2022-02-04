package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type OutgoingPaymentTestSuite struct {
	suite.Suite
}

func (OutgoingPaymentTestSuite) SetupSuite() {

}

func (OutgoingPaymentTestSuite) TearDownSuite() {

}

func (OutgoingPaymentTestSuite) TestAddInvoice() {

}

func TestOutgoingPaymentTestSuite(t *testing.T) {
	suite.Run(t, new(OutgoingPaymentTestSuite))
}
