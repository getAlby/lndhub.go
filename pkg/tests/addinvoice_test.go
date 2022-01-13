package tests

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type AddInvoiceTestSuite struct {
	suite.Suite
}

func (AddInvoiceTestSuite) SetupSuite() {

}

func (AddInvoiceTestSuite) TearDownSuite() {

}

func (AddInvoiceTestSuite) TestAddInvoice() {

}

func TestAddInvoiceTestSuite(t *testing.T) {
	suite.Run(t, new(AddInvoiceTestSuite))
}
