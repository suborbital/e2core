package local

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/schollz/peerdiscovery"

	"github.com/suborbital/e2core/foundation/bus/bus"
	"github.com/suborbital/vektor/vlog"
)

// Discovery is a grav Discovery plugin using local network multicast
type Discovery struct {
	opts     *bus.DiscoveryOpts
	log      *vlog.Logger
	stopChan chan struct{}

	discoveryFunc bus.DiscoveryFunc
}

// payload is a discovery payload
type payload struct {
	UUID string `json:"uuid"`
	Port string `json:"port"`
	Path string `json:"path"`
}

// New creates a new local discovery plugin
func New() *Discovery {
	g := &Discovery{}

	return g
}

// Start starts discovery
func (d *Discovery) Start(opts *bus.DiscoveryOpts, discoveryFunc bus.DiscoveryFunc) error {
	d.opts = opts
	d.log = opts.Logger
	d.discoveryFunc = discoveryFunc
	d.stopChan = make(chan struct{})

	d.log.Debug("[discovery-local] starting discovery, advertising endpoint", opts.TransportPort, opts.TransportURI)

	payloadFunc := func() []byte {
		payload := payload{
			UUID: d.opts.NodeUUID,
			Port: opts.TransportPort,
			Path: opts.TransportURI,
		}

		payloadBytes, _ := json.Marshal(payload)
		return payloadBytes
	}

	notifyFunc := func(peer peerdiscovery.Discovered) {
		d.log.Debug("[discovery-local] potential peer found:", peer.Address)

		payload := payload{}
		if err := json.Unmarshal(peer.Payload, &payload); err != nil {
			d.log.Debug("[discovery-local] peer did not offer correct payload, discarding")
			return
		}

		endpoint := fmt.Sprintf("%s:%s%s", peer.Address, payload.Port, payload.Path)

		// send the discovery to bus. Grav is responsible for ensuring uniqueness of the connections.
		d.discoveryFunc(endpoint, payload.UUID)
	}

	_, err := peerdiscovery.Discover(peerdiscovery.Settings{
		Limit:       -1,
		PayloadFunc: payloadFunc,
		Delay:       10 * time.Second,
		TimeLimit:   -1,
		Notify:      notifyFunc,
		AllowSelf:   true,
		StopChan:    d.stopChan,
	})

	return err
}

// UseDiscoveryFunc sets the function to be used when a new peer is discovered
func (d *Discovery) UseDiscoveryFunc(dFunc func(endpoint string, uuid string)) {
	d.discoveryFunc = dFunc
}

// Stop stops Discovery
func (d *Discovery) Stop() error {
	d.stopChan <- struct{}{}

	return nil
}
