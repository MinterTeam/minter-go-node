package statistics

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tendermint/tendermint/rpc/core"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
	"net"
	"net/url"
	"runtime"
	"sync"
	"time"
)

type ping struct {
	duration time.Duration
	url      string
}

func (d *Data) Statistic(ctx context.Context) {
	go d.handleStartBlocks(ctx)
	go d.handleEndBlock(ctx)
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
			wg.Add(state.NPeers)
			c := make(chan *ping, state.NPeers)
			for _, peer := range state.Peers {
				parse, err := url.Parse(peer.NodeInfo.ListenAddr)
				if err != nil {
					continue
				}
				go func(s string) {
					defer wg.Done()
					duration, err := pingTCP(s)
					if err != nil {
						return
					}
					c <- &ping{
						duration: duration,
						url:      s,
					}
				}(net.JoinHostPort(peer.RemoteIP, parse.Port()))
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

func pingTCP(url string) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", url, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return time.Since(start), nil
}

type Data struct {
	BlockStart struct {
		sync.RWMutex
		height          int64
		time            time.Time
		headerTimestamp time.Time
	}
	BlockEnd blockEnd
	cS       chan *StartRequest
	cE       chan *EndRequest
	Speed    struct {
		sync.RWMutex
		startTime         time.Time
		startHeight       int64
		duration          int64
		timerMin          <-chan time.Time
		blocksCountPerMin int64
		avgTimePerBlock   int64
	}

	Api  apiResponseTime
	Peer peerPing
}

type StartRequest struct {
	Height     int64
	Now        time.Time
	HeaderTime time.Time
}
type EndRequest struct {
	TimeEnd time.Time
	Height  int64
}

type LastBlockInfo struct {
	Height          int64
	Duration        int64
	HeaderTimestamp time.Time
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
		cS:       make(chan *StartRequest, 120),
		cE:       make(chan *EndRequest, 120),
	}
}

func (d *Data) PushStartBlock(req *StartRequest) {
	if d == nil {
		return
	}
	d.cS <- req
}

func (d *Data) handleStartBlocks(ctx context.Context) {
	if d == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-d.cS:
			func() {
				height := req.Height
				now := req.Now
				headerTime := req.HeaderTime

				for {
					d.BlockStart.RLock()
					ok := (height == d.BlockStart.height+1) || 0 == d.BlockStart.height
					d.BlockStart.RUnlock()
					if ok {
						break
					}
					runtime.Gosched()
				}

				d.BlockStart.Lock()
				defer d.BlockStart.Unlock()

				d.BlockStart.height = height
				d.BlockStart.time = now
				d.BlockStart.headerTimestamp = headerTime
			}()
		}
	}
}

func (d *Data) PushEndBlock(req *EndRequest) {
	if d == nil {
		return
	}
	d.cE <- req
}

func (d *Data) handleEndBlock(ctx context.Context) {
	if d == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-d.cE:
			func() {
				height := req.Height
				timeEnd := req.TimeEnd

				for {
					d.BlockStart.RLock()
					d.BlockEnd.RLock()
					ok := height == d.BlockStart.height
					d.BlockEnd.RUnlock()
					d.BlockStart.RUnlock()
					if ok {
						break
					}
					runtime.Gosched()
				}

				d.BlockStart.Lock()
				defer d.BlockStart.Unlock()

				duration := timeEnd.Sub(d.BlockStart.time)

				d.BlockEnd.Lock()
				defer d.BlockEnd.Unlock()

				d.BlockEnd.HeightProm.Set(float64(height))
				d.BlockEnd.DurationProm.Set(duration.Seconds())
				d.BlockEnd.TimestampProm.Set(float64(d.BlockStart.headerTimestamp.UnixNano()))

				d.BlockEnd.LastBlockInfo.Height = height
				d.BlockEnd.LastBlockInfo.Duration = duration.Nanoseconds()
				d.BlockEnd.LastBlockInfo.HeaderTimestamp = d.BlockStart.headerTimestamp

				d.Speed.Lock()
				defer d.Speed.Unlock()

				min := time.Minute
				select {
				case <-d.Speed.timerMin:
					d.Speed.avgTimePerBlock = int64(min) / d.Speed.blocksCountPerMin
					d.Speed.timerMin = time.After(min)
					d.Speed.blocksCountPerMin = 1
				default:
					d.Speed.blocksCountPerMin++
				}

				if time.Since(d.Speed.startTime) < 24*time.Hour {
					d.Speed.duration += duration.Nanoseconds()
					return
				}

				if d.Speed.startHeight == 0 {
					d.Speed.startTime = time.Now()
					d.Speed.startHeight = height
					d.Speed.duration = duration.Nanoseconds()
					d.Speed.timerMin = time.After(min)
					return
				}

				d.Speed.startTime = time.Now().Add(-12 * time.Hour)
				d.Speed.startHeight = height - (height-d.Speed.startHeight)/2
				d.Speed.duration = d.Speed.duration/2 + duration.Nanoseconds()

				return
			}()
		}
	}
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

func (d *Data) GetAverageBlockProcessingTime() int64 {
	if d == nil {
		return 0
	}

	d.Speed.RLock()
	defer d.Speed.RUnlock()

	d.BlockEnd.RLock()
	defer d.BlockEnd.RUnlock()

	if d.BlockEnd.LastBlockInfo.Height == d.Speed.startHeight {
		return 0
	}

	return d.Speed.duration / (d.BlockEnd.LastBlockInfo.Height - d.Speed.startHeight)
}

func (d *Data) GetTimePerBlock() int64 {
	if d == nil {
		return 0
	}

	d.Speed.RLock()
	defer d.Speed.RUnlock()

	return d.Speed.avgTimePerBlock
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
