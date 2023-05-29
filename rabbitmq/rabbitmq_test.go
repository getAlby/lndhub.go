package rabbitmq_test

import (
	"testing"
)

//go:generate mockgen -destination=./rabbitmqmocks/rabbitmq.go -package rabbitmqmocks github.com/getAlby/lndhub.go/rabbitmq LndHubService

func TestFinalizedInitializedPayments(t *testing.T) {
	t.Parallel()
}
