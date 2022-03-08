package plugin

import "github.com/getAlby/lndhub.go/lib/service"

func ProcessBalanceResponse(in int64, svc *service.LndhubService) (int64, error) {
	return 10 * in, nil
}
