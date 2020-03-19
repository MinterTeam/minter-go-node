package statistics

import (
	"context"
	"crypto/tls"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/tendermint/rpc/core"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"
)

func (d *Data) Statistic(ctx context.Context) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 10): //todo embedded period
			state, err := core.NetInfo(&rpctypes.Context{})
			if err != nil {
				continue
			}

			var wg sync.WaitGroup
			wg.Add(len(state.Peers))
			d.Peer.Lock()
			for _, peer := range state.Peers {
				u := &url.URL{Scheme: "http", Host: peer.RemoteIP}
				go func() {
					defer wg.Done()
					s := u.String()
					duration, err := timeGet(s)
					if err != nil {
						return
					}
					d.SetPeerTime(duration, s)
				}()
			}
			d.Peer.ping.Reset()
			d.Peer.Unlock()
			wg.Wait()
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

type Data struct {
	BlockStart struct {
		sync.Mutex
		height    uint64
		time      time.Time
		timestamp float64
	}
	BlockEnd blockEnd

	Api  apiResponseTime
	Peer peerPing
}
type blockEnd struct {
	sync.Mutex
	Height    prometheus.Gauge
	Duration  prometheus.Gauge
	Timestamp prometheus.Gauge
}
type apiResponseTime struct {
	sync.Mutex
	responseTime *prometheus.GaugeVec
}
type peerPing struct {
	sync.Mutex
	ping *prometheus.GaugeVec
}

func New() *Data {
	apiVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "api",
			Help: "Api Duration Paths",
		},
		[]string{"path"},
	)
	prometheus.MustRegister(apiVec)
	peerVec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "peers",
			Help: "Ping to Peers",
		},
		[]string{"network"},
	)
	prometheus.MustRegister(peerVec)
	lastBlockDuration := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "last_block_duration",
			Help: "Last block duration",
		},
	)
	prometheus.MustRegister(lastBlockDuration)
	height := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "height",
			Help: "Current height",
		},
	)
	prometheus.MustRegister(height)
	timeBlock := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "last_block_timestamp",
			Help: "Timestamp last block",
		},
	)
	prometheus.MustRegister(timeBlock)

	return &Data{
		Api:      apiResponseTime{responseTime: apiVec},
		Peer:     peerPing{ping: peerVec},
		BlockEnd: blockEnd{Height: height, Duration: lastBlockDuration, Timestamp: timeBlock},
	}
}

func (d *Data) SetStartBlock(height uint64, now time.Time, headerTime time.Time) {
	if d == nil {
		return
	}

	d.BlockStart.Lock()
	defer d.BlockStart.Unlock()

	d.BlockStart.height = height
	d.BlockStart.time = now
	d.BlockStart.timestamp = float64(headerTime.UnixNano() / 1e09)
}

func (d *Data) SetEndBlockDuration(timeEnd time.Time, height uint64) {
	if d == nil {
		return
	}

	d.BlockStart.Lock()
	defer d.BlockStart.Unlock()

	if height == d.BlockStart.height {
		d.BlockEnd.Lock()
		defer d.BlockEnd.Unlock()
		d.BlockEnd.Height.Set(float64(height))
		d.BlockEnd.Duration.Set(timeEnd.Sub(d.BlockStart.time).Seconds())
		d.BlockEnd.Timestamp.Set(d.BlockStart.timestamp)
		return
	}

	return
}

func (d *Data) SetApiTime(duration time.Duration, path string) {
	if d == nil {
		return
	}

	d.Api.Lock()
	defer d.Api.Unlock()

	d.Api.responseTime.With(prometheus.Labels{"path": path}).Set(duration.Seconds())
}

func (d *Data) SetPeerTime(duration time.Duration, network string) {
	if d == nil {
		return
	}

	d.Peer.Lock()
	defer d.Peer.Unlock()

	d.Peer.ping.With(prometheus.Labels{"network": network}).Set(duration.Seconds())
}
