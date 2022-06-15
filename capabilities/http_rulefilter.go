package capabilities

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

var (
	ErrHttpDisallowed    = errors.New("requests to insecure HTTP endpoints is disallowed")
	ErrIPsDisallowed     = errors.New("requests to IP addresses are disallowed")
	ErrPrivateDisallowed = errors.New("requests to private IP address ranges are disallowed")
	ErrDomainDisallowed  = errors.New("requests to this domain are disallowed")
	ErrPortDisallowed    = errors.New("requests to this port are disallowed")
)

// HTTPRules is a set of rules that governs use of the HTTP capability
type HTTPRules struct {
	AllowedDomains []string `json:"allowedDomains" yaml:"allowedDomains"`
	BlockedDomains []string `json:"blockedDomains" yaml:"blockedDomains"`
	AllowedPorts   []int    `json:"allowedPorts" yaml:"allowedPorts"`
	BlockedPorts   []int    `json:"blockedPorts" yaml:"blockedPorts"`
	AllowIPs       bool     `json:"allowIPs" yaml:"allowIPs"`
	AllowPrivate   bool     `json:"allowPrivate" yaml:"allowPrivate"`
	AllowHTTP      bool     `json:"allowHTTP" yaml:"allowHTTP"`
}

var standardPorts = []int{80, 443}

// requestIsAllowed returns a non-nil error if the provided request is not allowed to proceed
func (h HTTPRules) requestIsAllowed(req *http.Request) error {
	// Hostname removes port numbers as well as IPv6 [ and ]
	hosts := []string{req.URL.Hostname()}

	if !h.AllowHTTP {
		if req.URL.Scheme == "http" {
			return ErrHttpDisallowed
		}
	}

	// Evaluate port access rules
	if err := h.portAllowed(req.URL); err != nil {
		return err
	}

	// determine if the passed-in host is an IP address
	isRawIP := net.ParseIP(req.URL.Hostname()) != nil
	if !h.AllowIPs && isRawIP {
		return ErrIPsDisallowed
	}

	// determine if the host is a CNAME record and resolve it
	// to be checked in addition to the passed-in raw host
	resolvedCNAME, err := net.LookupCNAME(req.URL.Host)
	if err != nil {
		// that's ok, it just means there is no CNAME
	} else if resolvedCNAME != "" && resolvedCNAME != req.URL.Host {
		hosts = append(hosts, resolvedCNAME)
	}

	allowDefault := true // if neither allowed or blocked domains are configured, the default is to allow

	for _, host := range hosts {
		// first check for resolved private IPs if needed
		if !h.AllowPrivate {
			if err := resolvesToPrivate(host); err != nil {
				return err
			}
		}

		// if AllowedDomains are listed, they take precednece over BlockedDomains
		// (an explicit allowlist is more strict than a blocklist, so we default to that)
		if len(h.AllowedDomains) > 0 {
			allowDefault = false

			// check each allowed domain, and if any match, return nil
			for _, d := range h.AllowedDomains {
				if matchesDomain(d, host) {
					return nil
				}
			}
		} else if len(h.BlockedDomains) > 0 {
			allowDefault = true

			// check each blocked domain, if any match return an error
			for _, d := range h.BlockedDomains {
				if matchesDomain(d, host) {
					return ErrDomainDisallowed
				}
			}
		}

		if !allowDefault {
			return ErrDomainDisallowed
		}
	}

	return nil
}

// portAllowed evaluates port allowance rules
func (h HTTPRules) portAllowed(url *url.URL) error {
	// Backward Compatibility:
	// Allow all ports if no allow/block list has been configured
	if len(h.AllowedPorts)+len(h.BlockedPorts) == 0 {
		return nil
	}

	port, err := readPort(url)
	if err != nil {
		return ErrPortDisallowed
	}

	if slices.Contains(h.BlockedPorts, port) {
		return ErrPortDisallowed
	}

	for _, p := range append(standardPorts, h.AllowedPorts...) {
		if p == port {
			return nil
		}
	}

	return ErrPortDisallowed
}

// readPort returns normalized URL port
func readPort(url *url.URL) (int, error) {
	if url.Port() == "" {
		if url.Scheme == "https" {
			return 443, nil
		}
		return 80, nil
	}

	return strconv.Atoi(url.Port())
}

// returns nil if the host does not resolve to an IP in a private range
// returns ErrPrivateDisallowed if it does
func resolvesToPrivate(host string) error {
	if strings.Contains(host, "localhost") {
		return ErrPrivateDisallowed
	}

	// resolve DNS before checking
	ips, err := net.LookupIP(host)
	if err != nil {
		dnsErr, isDNSErr := err.(*net.DNSError)
		if isDNSErr && dnsErr.IsNotFound {
			// that's ok, let things continue even if the host does not resolve
		} else {
			return errors.Wrap(err, "failed to LookupIP")
		}
	}

	for _, ip := range ips {
		if ip.IsPrivate() || !ip.IsGlobalUnicast() {
			return ErrPrivateDisallowed
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
		if domainParts[i] == "" {
			// skip over empty members
			// of the domain being checked
			i--
		}

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
		AllowPrivate:   true,
	}

	return h
}
