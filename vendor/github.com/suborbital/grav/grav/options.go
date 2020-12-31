package grav

import "github.com/suborbital/vektor/vlog"

// Options represent Grav options
type Options struct {
	Logger    *vlog.Logger
	Transport Transport
	Discovery Discovery
	Port      string
	URI       string
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

// UseTransport sets the transport plugin to be used.
func UseTransport(transport Transport) OptionsModifier {
	return func(o *Options) {
		o.Transport = transport
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

func defaultOptions() *Options {
	o := &Options{
		Logger:    vlog.Default(),
		Port:      "8080",
		URI:       "/meta/message",
		Transport: nil,
		Discovery: nil,
	}

	return o
}
