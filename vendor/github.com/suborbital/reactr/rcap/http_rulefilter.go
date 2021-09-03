package rcap

import (
	"net"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrHttpDisallowed   = errors.New("requests to insecure HTTP endpoints is disallowed")
	ErrIPsDisallowed    = errors.New("requests to IP addresses are disallowed")
	ErrDomainDisallowed = errors.New("requests to this domain are disallowed")
)

// HTTPRules is a set of rules that governs use of the HTTP capability
type HTTPRules struct {
	AllowedDomains []string `json:"allowedDomains" yaml:"allowedDomains"`
	BlockedDomains []string `json:"blockedDomains" yaml:"blockedDomains"`
	AllowIPs       bool     `json:"allowIPs" yaml:"allowIPs"`
	AllowHTTP      bool     `json:"allowHTTP" yaml:"allowHTTP"`
}

// requestIsAllowed returns a non-nil error if the provided request is not allowed to proceed
func (h HTTPRules) requestIsAllowed(req *http.Request) error {
	if !h.AllowHTTP {
		if req.URL.Scheme == "http" {
			return ErrHttpDisallowed
		}
	}

	if !h.AllowIPs {
		if net.ParseIP(req.URL.Host) != nil {
			return ErrIPsDisallowed
		}
	}

	// if AllowedDomains are listed, they take precednece over BlockedDomains
	// (an explicit allowlist is more strict than a blocklist, so we default to that)
	if len(h.AllowedDomains) > 0 {
		// check each allowed domain, and if any match, return nil
		for _, d := range h.AllowedDomains {
			if matchesDomain(d, req.URL.Host) {
				return nil
			}
		}

		return ErrDomainDisallowed
	} else if len(h.BlockedDomains) > 0 {
		// check each blocked domain, if any match return an error
		for _, d := range h.BlockedDomains {
			if matchesDomain(d, req.URL.Host) {
				return ErrDomainDisallowed
			}
		}
	}

	return nil
}

func matchesDomain(pattern, domain string) bool {
	if pattern == "" && domain == "" {
		return true
	}

	if pattern == "" || domain == "" {
		return false
	}

	if pattern == domain {
		return true
	}

	patternParts := strings.Split(pattern, ".")
	domainParts := strings.Split(domain, ".")

	if len(patternParts) > len(domainParts) {
		return false
	}

	// iterate over the pattern and domain *backwards* to determine
	// if the domain matches the pattern with wildcard support
	j := len(patternParts) - 1
	for i := len(domainParts) - 1; i >= 0; i-- {
		p := patternParts[j]
		d := domainParts[i]

		if p == "*" || p == d {
			// do nothing, they match
		} else {
			return false
		}

		if j > 0 {
			j--
		}
	}

	return true
}

// defaultHTTPRules returns the default rules with all requests allowed
func defaultHTTPRules() HTTPRules {
	h := HTTPRules{
		AllowedDomains: []string{},
		BlockedDomains: []string{},
		AllowIPs:       true,
		AllowHTTP:      true,
	}

	return h
}
