package controllers

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/skip2/go-qrcode"
)

const (
	SAT_PER_BTC = 1e8
)

// HomeController : HomeController struct
type HomeController struct {
	svc  *service.LndhubService
	html string
}

func NewHomeController(svc *service.LndhubService, html string) *HomeController {
	return &HomeController{
		svc:  svc,
		html: html,
	}
}

type HomepageContent struct {
	NumActiveChannels  int
	NumPendingChannels int
	NumPeers           int
	SyncedToChain      bool
	BlockHeight        int
	Uris               []string
	Channels           []Channel
	Branding           service.BrandingConfig
}

type Channel struct {
	Name         string
	RemotePubkey string
	CapacityBTC  float64
	Local        int
	Total        int
	Size         int
	Active       bool
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}
func (controller *HomeController) QR(c echo.Context) error {
	customPath := strings.Replace(c.Request().URL.Path, "/qr", "", 1)
	encoded := url.QueryEscape(fmt.Sprintf("%s://%s%s", c.Request().URL.Scheme, c.Request().Host, customPath))
	url := fmt.Sprintf("bluewallet:setlndhuburl?url=%s", encoded)
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		return err
	}
	return c.Blob(http.StatusOK, "image/png", png)
}

func (controller *HomeController) Home(c echo.Context) error {
	info, err := controller.svc.GetInfo(c.Request().Context())
	if err != nil {
		return err
	}
	channels, err := controller.svc.LndClient.ListChannels(c.Request().Context(), &lnrpc.ListChannelsRequest{})
	if err != nil {
		return err
	}

	tmpl, err := template.New("index").Parse(controller.html)
	if err != nil {
		return err
	}
	// See original code: https://github.com/BlueWallet/LndHub/blob/master/controllers/website.js#L32
	maxChanCapacity := -1
	for _, ch := range channels.Channels {
		maxChanCapacity = Max(maxChanCapacity, int(ch.Capacity))
	}
	channelSlice := []Channel{}
	for _, ch := range channels.Channels {

		magic := maxChanCapacity / 100
		channelSlice = append(channelSlice, Channel{
			Name:         pubkeyToName[ch.RemotePubkey],
			RemotePubkey: ch.RemotePubkey,
			CapacityBTC:  float64(ch.Capacity) / SAT_PER_BTC,
			Local:        int(ch.LocalBalance),
			Total:        int(ch.Capacity),
			Size:         int(ch.Capacity) / magic,
			Active:       ch.Active,
		})
	}
	content := HomepageContent{
		NumActiveChannels:  int(info.NumActiveChannels),
		NumPendingChannels: int(info.NumPendingChannels),
		NumPeers:           int(info.NumPeers),
		SyncedToChain:      info.SyncedToChain,
		BlockHeight:        int(info.BlockHeight),
		Channels:           channelSlice,
		Uris:               info.Uris,
		Branding:           controller.svc.Config.Branding,
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, content)
	if err != nil {
		return err
	}
	c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=300, stale-if-error=21600") // cache for 5 minutes or if error for 6 hours max
	return c.HTMLBlob(http.StatusOK, buf.Bytes())
}

var pubkeyToName = map[string]string{
	"03e50492eab4107a773141bb419e107bda3de3d55652e6e1a41225f06a0bbf2d56": "yalls.org",
	"0232e20e7b68b9b673fb25f48322b151a93186bffe4550045040673797ceca43cf": "zigzag.io",
	"02df5ffe895c778e10f7742a6c5b8a0cefbe9465df58b92fadeb883752c8107c8f": "blockstream store",
	"030c3f19d742ca294a55c00376b3b355c3c90d61c6b6b39554dbc7ac19b141c14f": "bitrefill.com",
	"03864ef025fde8fb587d989186ce6a4a186895ee44a926bfc370e2c366597a3f8f": "ACINQ",
	"03abf6f44c355dec0d5aa155bdbdd6e0c8fefe318eff402de65c6eb2e1be55dc3e": "OpenNode",
	"028d98b9969fbed53784a36617eb489a59ab6dc9b9d77fcdca9ff55307cd98e3c4": "OpenNode 2",
	"0242a4ae0c5bef18048fbecf995094b74bfb0f7391418d71ed394784373f41e4f3": "coingate.com",
	"0279c22ed7a068d10dc1a38ae66d2d6461e269226c60258c021b1ddcdfe4b00bc4": "ln1.satoshilabs.com",
	"02c91d6aa51aa940608b497b6beebcb1aec05be3c47704b682b3889424679ca490": "lnd-21.LNBIG.com",
	"024655b768ef40951b20053a5c4b951606d4d86085d51238f2c67c7dec29c792ca": "satoshis.place",
	"03c2abfa93eacec04721c019644584424aab2ba4dff3ac9bdab4e9c97007491dda": "tippin.me",
	"022c699df736064b51a33017abfc4d577d133f7124ac117d3d9f9633b6297a3b6a": "globee.com",
	"0237fefbe8626bf888de0cad8c73630e32746a22a2c4faa91c1d9877a3826e1174": "1.ln.aantonop.com",
	"026c7d28784791a4b31a64eb34d9ab01552055b795919165e6ae886de637632efb": "LivingRoomOfSatoshi",
	"02816caed43171d3c9854e3b0ab2cf0c42be086ff1bd4005acc2a5f7db70d83774": "ln.pizza",
	"0254ff808f53b2f8c45e74b70430f336c6c76ba2f4af289f48d6086ae6e60462d3": "bitrefill thor",
	"03d607f3e69fd032524a867b288216bfab263b6eaee4e07783799a6fe69bb84fac": "bitrefill 3",
	"02a0bc43557fae6af7be8e3a29fdebda819e439bea9c0f8eb8ed6a0201f3471ca9": "LightningPeachHub",
	"02d4531a2f2e6e5a9033d37d548cff4834a3898e74c3abe1985b493c42ebbd707d": "coinfinity.co",
	"02d23fa6794d8fd056c757f3c8f4877782138dafffedc831fc570cab572620dc61": "paywithmoon.com",
	"025f1456582e70c4c06b61d5c8ed3ce229e6d0db538be337a2dc6d163b0ebc05a5": "paywithmoon.com",
	"02004c625d622245606a1ea2c1c69cfb4516b703b47945a3647713c05fe4aaeb1c": "walletofsatoshi",
	"0331f80652fb840239df8dc99205792bba2e559a05469915804c08420230e23c7c": "LightningPowerUsers.com",
	"033d8656219478701227199cbd6f670335c8d408a92ae88b962c49d4dc0e83e025": "bfx-lnd0",
	"03021c5f5f57322740e4ee6936452add19dc7ea7ccf90635f95119ab82a62ae268": "lnd1.bluewallet.io",
	"037cc5f9f1da20ac0d60e83989729a204a33cc2d8e80438969fadf35c1c5f1233b": "lnd2.bluewallet.io",
	"036b53093df5a932deac828cca6d663472dbc88322b05eec1d42b26ab9b16caa1c": "okcoin",
	"038f8f113c580048d847d6949371726653e02b928196bad310e3eda39ff61723f6": "magnetron",
	"03829249ef39746fd534a196510232df08b83db0967804ec71bf4120930864ff97": "blokada.org",
	"02ce691b2e321954644514db708ba2a72769a6f9142ac63e65dd87964e9cf2add9": "Satoshis.Games",
}
