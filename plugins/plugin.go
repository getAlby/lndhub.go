package plugin

import (
	"os"
	"reflect"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func CreatePlugin(path, funcName string) (res reflect.Value, err error) {
	//todo: we can fetch the source from a url
	src, err := os.ReadFile(path)
	if err != nil {
		return res, err
	}
	intp := interp.New(interp.Options{})
	err = intp.Use(stdlib.Symbols)
	if err != nil {
		return res, err
	}
	err = intp.Use(map[string]map[string]reflect.Value{
		"github.com/getAlby/lndhub.go/lib/service/service": {
			"LndhubService": reflect.ValueOf((*service.LndhubService)(nil)),
		},
		"github.com/getAlby/lndhub.go/db/models/models": {
			"User": reflect.ValueOf((*models.User)(nil)),
		},
	})
	if err != nil {
		return res, err
	}
	_, err = intp.Eval(string(src))
	if err != nil {
		return res, err
	}
	return intp.Eval(funcName)
}
