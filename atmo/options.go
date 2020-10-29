package atmo

import "github.com/suborbital/vektor/vk"

// Options defines options for Atmo
type Options struct {
	vkOpts vk.Options
}

// OptionModifier defines options for Atmo
type OptionModifier func(Options) Options
