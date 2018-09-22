package addons

import (
	"github.com/yittg/ving/net"
	"github.com/yittg/ving/net/protocol"
	"github.com/yittg/ving/options"
)

// AddOn features
type AddOn interface {
	Init([]*protocol.NetworkTarget, chan bool, *options.Option, *net.NPing)

	Start()

	Collect()

	Activate()

	Deactivate()

	RenderState() interface{}

	NewUI() UI
}
