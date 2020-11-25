package grav

import "github.com/suborbital/vektor/vlog"

// Options represent Grav options
type Options struct {
	Logger    *vlog.Logger
	Port      string
	Transport Transport
	Discovery Discovery
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
func UseLogger(logger *vlog.Logger) OptionsModifier {
	return func(o *Options) {
		o.Logger = logger
	}
}

// UsePort sets the port that will be advertised by discovery
func UsePort(port string) OptionsModifier {
	return func(o *Options) {
		o.Port = port
	}
}

// UseTransport sets the transport plugin to be used
func UseTransport(transport Transport) OptionsModifier {
	return func(o *Options) {
		o.Transport = transport
	}
}

// UseDiscovery sets the discovery plugin to be used
func UseDiscovery(discovery Discovery) OptionsModifier {
	return func(o *Options) {
		o.Discovery = discovery
	}
}

func defaultOptions() *Options {
	o := &Options{
		Logger:    vlog.Default(),
		Port:      "8080",
		Transport: nil,
		Discovery: nil,
	}

	return o
}
