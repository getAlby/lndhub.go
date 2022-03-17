package plugin

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func LoadMiddlewarePlugins(pluginList string) (res map[string]reflect.Value, err error) {
	res = map[string]reflect.Value{}
	pluginSlice := strings.Split(pluginList, ",")
	for _, url := range pluginSlice {
		endpoint, err := extractMiddlewareEndpoint(url)
		plug, err := CreateMiddlewarePlugin(url, "plugin.Middleware")
		if err != nil {
			return nil, err
		}
		res[endpoint] = plug
	}
	return res, nil
}

func extractMiddlewareEndpoint(pluginUrl string) (result string, err error) {
	//the middleware plugin url should have the following structure: https://example.com/path/to/your/middleware_<endpoint>_<name>.go
	u, err := url.Parse(pluginUrl)
	if err != nil {
		return "", err
	}
	parts := strings.Split(path.Base(u.Path), "_")
	if len(parts) != 3 {
		return "", fmt.Errorf("Invalid format %s", pluginUrl)
	}
	return parts[1], nil
}

func CreateMiddlewarePlugin(url, funcName string) (res reflect.Value, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return res, err
	}
	src, err := ioutil.ReadAll(resp.Body)
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
