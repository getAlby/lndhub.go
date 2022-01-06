package tests

import (
	"github.com/stretchr/testify/suite"
	"testing"
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
