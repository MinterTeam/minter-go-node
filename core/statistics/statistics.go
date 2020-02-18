package statistics

import (
	"crypto/tls"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/tendermint/tendermint/rpc/core"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"
)

func Statistic(app *minter.Blockchain) {
	height, durationBlock := app.GetLastBlockDuration()
	log.Println(height, durationBlock)
	log.Println(app.GetApiTime())

	state, err := core.NetInfo(&rpctypes.Context{})
	if err != nil {
		log.Fatalln(err)
	}
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	for _, peer := range state.Peers {
		u := &url.URL{
			Scheme:     "http",
			Opaque:     "",
			User:       nil,
			Host:       peer.RemoteIP,
			Path:       "",
			RawPath:    "",
			ForceQuery: false,
			RawQuery:   "",
			Fragment:   "",
		}
		log.Println(u.String(), timeGet(u.String()))
	}

}

func timeGet(url string) time.Duration {
	req, _ := http.NewRequest("GET", url, nil)

	var start time.Time

	trace := &httptrace.ClientTrace{}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	start = time.Now()
	if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
		log.Println(err)
	}
	return time.Since(start)
}
