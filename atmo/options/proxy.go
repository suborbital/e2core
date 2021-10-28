//go:build !proxy

package options

func proxyEnabled() bool {
	return false
}
