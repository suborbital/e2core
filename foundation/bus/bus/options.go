package bus

import (
	"os"

	"github.com/rs/zerolog"
)

// Options represent Bus options
type Options struct {
	Logger          zerolog.Logger
	MeshTransport   MeshTransport
	BridgeTransport BridgeTransport
	Discovery       Discovery
	Port            string
	URI             string
	BelongsTo       string
	Interests       []string
}

// OptionsModifier is function that modifies an option
type OptionsModifier func(*Options)

func newOptionsWithModifiers(mods ...OptionsModifier) *Options {
	opts := defaultOptions()

	for _, m := range mods {
		m(opts)
	}

	return opts
}

// UseLogger allows a custom logger to be used
func UseLogger(logger zerolog.Logger) OptionsModifier {
	return func(o *Options) {
		o.Logger = logger
	}
}

// UseMeshTransport sets the mesh transport plugin to be used.
func UseMeshTransport(mesh MeshTransport) OptionsModifier {
	return func(o *Options) {
		o.MeshTransport = mesh
	}
}

// UseBridgeTransport sets the mesh transport plugin to be used.
func UseBridgeTransport(bridge BridgeTransport) OptionsModifier {
	return func(o *Options) {
		o.BridgeTransport = bridge
	}
}

// UseEndpoint sets the endpoint settings for the instance to broadcast for discovery
// Pass empty strings for either if you would like to keep the defaults (8080 and /meta/message)
func UseEndpoint(port, uri string) OptionsModifier {
	return func(o *Options) {
		if port != "" {
			o.Port = port
		}

		if uri != "" {
			o.URI = uri
		}
	}
}

// UseDiscovery sets the discovery plugin to be used
func UseDiscovery(discovery Discovery) OptionsModifier {
	return func(o *Options) {
		o.Discovery = discovery
	}
}

// UseBelongsTo sets the 'BelongsTo' property for the Grav instance
func UseBelongsTo(belongsTo string) OptionsModifier {
	return func(o *Options) {
		o.BelongsTo = belongsTo
	}
}

// UseInterests sets the 'Interests' property for the Grav instance
func UseInterests(interests ...string) OptionsModifier {
	return func(o *Options) {
		o.Interests = interests
	}
}

func defaultOptions() *Options {
	o := &Options{
		BelongsTo:       "*",
		Interests:       []string{},
		Logger:          zerolog.New(os.Stderr).With().Str("mode", "default-options").Timestamp().Logger(),
		Port:            "8080",
		URI:             "/meta/message",
		MeshTransport:   nil,
		BridgeTransport: nil,
		Discovery:       nil,
	}

	return o
}
