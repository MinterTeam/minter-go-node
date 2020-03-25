package statistics

import (
	"context"
	"crypto/tls"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/tendermint/rpc/core"
	rpctypes "github.com/tendermint/tendermint/rpc/lib/types"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"sync"
	"time"
)

type ping struct {
	duration time.Duration
	url      string
}

func (d *Data) Statistic(ctx context.Context) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 10):
			state, err := core.NetInfo(&rpctypes.Context{})
			if err != nil {
				continue
			}

			var wg sync.WaitGroup
			countPeers := len(state.Peers)
			wg.Add(countPeers)
			c := make(chan *ping, countPeers)
			for _, peer := range state.Peers {
				parse, err := url.Parse(peer.NodeInfo.ListenAddr)
				if err != nil {
					continue
				}
				u := &url.URL{Scheme: "http", Host: net.JoinHostPort(peer.RemoteIP, parse.Port())}
				go func(s string) {
					defer wg.Done()
					duration, err := timeGet(s)
					if err != nil {
						return
					}
					c <- &ping{
						duration: duration,
						url:      s,
					}
				}(u.String())
			}
			wg.Wait()
			d.ResetPeersPing()
			close(c)
			for ping := range c {
				d.SetPeerTime(ping.duration, ping.url)
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

type Data struct {
	BlockStart struct {
		sync.RWMutex
		height    uint64
		time      time.Time
		timestamp float64
	}
	BlockEnd blockEnd

	Api  apiResponseTime
	Peer peerPing
}

type LastBlockInfo struct {
	Height    uint64
	Duration  float64
	Timestamp float64
}

type blockEnd struct {
	sync.RWMutex
	HeightProm    prometheus.Gauge
	DurationProm  prometheus.Gauge
	TimestampProm prometheus.Gauge
	LastBlockInfo LastBlockInfo
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
			Help: "Api DurationProm Paths",
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
			Help: "TimestampProm last block",
		},
	)
	prometheus.MustRegister(timeBlock)

	return &Data{
		Api:      apiResponseTime{responseTime: apiVec},
		Peer:     peerPing{ping: peerVec},
		BlockEnd: blockEnd{HeightProm: height, DurationProm: lastBlockDuration, TimestampProm: timeBlock},
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

	d.BlockStart.RLock()
	defer d.BlockStart.RUnlock()

	if height == d.BlockStart.height {
		d.BlockEnd.Lock()
		defer d.BlockEnd.Unlock()

		durationSeconds := timeEnd.Sub(d.BlockStart.time).Seconds()

		d.BlockEnd.HeightProm.Set(float64(height))
		d.BlockEnd.DurationProm.Set(durationSeconds)
		d.BlockEnd.TimestampProm.Set(d.BlockStart.timestamp)

		d.BlockEnd.LastBlockInfo.Height = height
		d.BlockEnd.LastBlockInfo.Duration = durationSeconds
		d.BlockEnd.LastBlockInfo.Timestamp = d.BlockStart.timestamp

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

func (d *Data) GetLastBlockInfo() LastBlockInfo {
	if d == nil {
		return LastBlockInfo{}
	}

	d.BlockEnd.RLock()
	defer d.BlockEnd.RUnlock()

	return d.BlockEnd.LastBlockInfo
}

func (d *Data) ResetPeersPing() {
	if d == nil {
		return
	}

	d.Peer.Lock()
	defer d.Peer.Unlock()

	d.Peer.ping.Reset()
}
