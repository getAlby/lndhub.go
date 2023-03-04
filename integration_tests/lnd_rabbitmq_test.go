package integration_tests

import (
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/suite"
)

type LndRabbitMQSuite struct {
	TestSuite
	service *service.LndhubService
}

func (suite *LndRabbitMQSuite) SetupSuite() {

	mockLnd := newDefaultMockLND()

	svc, err := LndHubTestServiceInit(mockLnd)
	if assert.Error(suite.T(), err) {
		suite.T().Fail()
	}

	suite.service = svc
}

func (suite *LndRabbitMQSuite) TestDummy() {
	assert.NotNil(suite.T(), suite.service.Config.RabbitMQUri)
	suite.T().Log(suite.service.Config.RabbitMQUri)
}

func TestLndRabbitMQSuite(t *testing.T) {
	suite.Run(t, new(LndRabbitMQSuite))
}
