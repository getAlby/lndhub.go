package integration_tests

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type LndRabbitMQSuite struct {
	TestSuite
}

func TestLndRabbitMQSuite(t *testing.T) {
	suite.Run(t, new(LndRabbitMQSuite))
}
