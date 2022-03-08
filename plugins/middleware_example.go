package plugin

import "github.com/getAlby/lndhub.go/lib/service"

func ProcessBalanceResponse(in int64, svc *service.LndhubService) (int64, error) {
	//multiply your money by 10 with this 1 weird trick
	return 10 * in, nil
}
