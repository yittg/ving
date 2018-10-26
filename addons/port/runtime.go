package port

import (
	"strconv"
	"sync"
	"time"

	"github.com/yittg/ving/addons"
	"github.com/yittg/ving/addons/port/types"
	"github.com/yittg/ving/config"
	"github.com/yittg/ving/net"
	"github.com/yittg/ving/net/protocol"
	"github.com/yittg/ving/options"
)

type runtime struct {
	targets    []*protocol.NetworkTarget
	rawTargets []string
	stop       chan bool
	ping       *net.NPing
	opt        *options.Option
	active     bool

	selected    chan int
	resultChan  chan *touchResult
	refreshChan chan int

	targetPorts []types.PortDesc
	targetDone  sync.Map
	results     map[int][]touchResultWrapper

	proberPool     sync.Map
	proberPoolSize int

	ui         *ui
	initUILock sync.Once
}

type touchResult struct {
	id        int
	portID    int
	connected bool
	connTime  time.Duration
}

type touchResultWrapper struct {
	port types.PortDesc
	res  *touchResult
}

type prober struct {
	pipe    chan *probeUnit
	running sync.Once
}

type probeUnit struct {
	id     int
	portID int
	target *protocol.NetworkTarget
}

func newPortAddOn() addons.AddOn {
	portConfig := config.GetConfig().AddOns.Ports
	return &runtime{
		selected:       make(chan int, 1),
		proberPool:     sync.Map{},
		proberPoolSize: portConfig.ProbeConcurrency,
		resultChan:     make(chan *touchResult, 1024),
		targetDone:     sync.Map{},
		results:        make(map[int][]touchResultWrapper),
		refreshChan:    make(chan int, 1),
		stop:           make(chan bool, 2),
	}
}

// Desc of this port add-on
func (*runtime) Desc() string {
	return "port probe"
}

// Init ports scanner
func (rt *runtime) Init(envoy *addons.Envoy) {
	rt.targets = envoy.Targets
	rt.opt = envoy.Opt
	rt.ping = envoy.Ping
	for _, t := range rt.targets {
		rt.rawTargets = append(rt.rawTargets, t.Raw)
	}

	if len(rt.opt.MorePorts) > 0 {
		for _, p := range rt.opt.MorePorts {
			rt.targetPorts = append(rt.targetPorts, types.PortDesc{Name: strconv.Itoa(p), Port: p})
		}
		rt.opt.Ports = true
	} else {
		rt.targetPorts = getPredefinedPorts()
	}
}

func (rt *runtime) Start() {
	go rt.scanPorts()
}

func (rt *runtime) Stop() {
	close(rt.stop)
}

func (rt *runtime) scanPorts() {
	var selected int
	var host *protocol.NetworkTarget
	ticker := time.NewTicker(time.Millisecond * 10)
	for {
		select {
		case <-rt.stop:
			return
		case selected = <-rt.selected:
			if selected < 0 || selected >= len(rt.targets) {
				host = nil
				continue
			}
			host = rt.targets[selected]
		case id := <-rt.refreshChan:
			rt.selected <- id
		case <-ticker.C:
			if !rt.active || host == nil || !rt.checkStart(selected) {
				break
			}
			rt.targetDone.Store(selected, false)
			for i, port := range rt.targetPorts {
				rt.probeTargetAsyc(selected, i, protocol.TCPTarget(host, port.Port))
			}
			host = nil
		}
	}
}

func (rt *runtime) probeTargetAsyc(idx, portID int, t *protocol.NetworkTarget) {
	bucket := portID % rt.proberPoolSize
	_p, existed := rt.proberPool.LoadOrStore(bucket, &prober{})
	p := _p.(*prober)
	if !existed {
		p.pipe = make(chan *probeUnit, 100)
		p.running.Do(func() {
			go func(pipe chan *probeUnit) {
				for pu := range pipe {
					connTime, err := rt.ping.PingOnce(pu.target, time.Second)
					rt.resultChan <- &touchResult{
						id:        pu.id,
						portID:    pu.portID,
						connected: err == nil,
						connTime:  connTime,
					}
				}
			}(p.pipe)
		})
	}
	p.pipe <- &probeUnit{
		id:     idx,
		portID: portID,
		target: t,
	}
}

func (rt *runtime) resetTargetStatus(id int) {
	if !rt.checkDone(id) {
		return
	}

	rt.results[id] = rt.prepareTouchResults()
	rt.targetDone.Delete(id)
	rt.refreshChan <- id
}

func (rt *runtime) prepareTouchResults() []touchResultWrapper {
	s := make([]touchResultWrapper, len(rt.targetPorts))
	for i, port := range rt.targetPorts {
		s[i] = touchResultWrapper{
			port: port,
		}
	}
	return s
}

func (rt *runtime) Schedule() {
	updated := make(map[int]bool)
	for {
		select {
		case res := <-rt.resultChan:
			s, ok := rt.results[res.id]
			if !ok {
				s = rt.prepareTouchResults()
				rt.results[res.id] = s
			}
			s[res.portID].res = res
			updated[res.id] = true
		default:
			for id := range updated {
				done := true
				for _, s := range rt.results[id] {
					if s.res == nil {
						done = false
						break
					}
				}
				rt.targetDone.Store(id, done)
			}
			return
		}
	}
}

func (rt *runtime) updateStatus(active bool) {
	rt.active = active
}

func (rt *runtime) State() interface{} {
	return rt.results
}

// GetUI init a ui for this add-on
func (rt *runtime) GetUI() addons.UI {
	if rt.ui == nil {
		rt.initUILock.Do(func() {
			rt.ui = newUI(rt)
		})
	}
	return rt.ui
}

func (rt *runtime) checkStart(idx int) bool {
	_, ok := rt.targetDone.Load(idx)
	return !ok
}

func (rt *runtime) checkDone(idx int) bool {
	done, ok := rt.targetDone.Load(idx)
	return ok && done.(bool)
}
