package statistics

import (
	"context"
	"crypto/tls"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/tendermint/tendermint/rpc/core"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
	"log"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"
)

func Statistic(ctx context.Context, app *minter.Blockchain) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 300):
			state, err := core.NetInfo(&rpctypes.Context{})
			if err != nil {
				log.Fatalln(err)
			}
			var wg sync.WaitGroup
			wg.Add(len(state.Peers))
			for _, peer := range state.Peers {
				u := &url.URL{Scheme: "http", Host: peer.RemoteIP}
				func() {
					s := u.String()
					duration, err := timeGet(s)
					if err != nil {

					}
					app.SetPeerTime(duration, s)
				}()
			}
		}
	}
}

func timeGet(url string) (time.Duration, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{}))
	start := time.Now()
	_, err = http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return 0, err
	}
	return time.Since(start), nil
}
