package capabilities

import (
	"net/http"
	"testing"
)

func testRequestIsAllowed(t *testing.T, name string, rules HTTPRules, url string, shouldError bool) {
	t.Run(name, func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, url, nil)

		err := rules.requestIsAllowed(req)

		if shouldError && err == nil {
			t.Error("error did not occur, should have")
		} else if !shouldError && err != nil {
			t.Error("error occurred, should not have:", err)
		}
	})
}

func TestDefaultRules(t *testing.T) {
	rules := defaultHTTPRules()

	tests := []struct {
		name string
		url  string
	}{
		{"http allowed", "http://example.com"},
		{"https allowed", "https://example.com"},
		{"IP allowed", "http://100.11.12.13"},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, false)
	}
}

func TestAllowedDomains(t *testing.T) {
	rules := defaultHTTPRules()
	rules.AllowedDomains = []string{"example.com", "another.com", "*.hello.com", "tomorrow.*", "100.*.12.13"}

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"example.com:8080 allowed", "http://example.com:8080", false},
		{"example.com allowed", "http://example.com", false},
		{"another.com allowed", "http://another.com", false},
		{"wildcard allowed", "http://goodbye.hello.com", false},
		{"double wildcard allowed", "http://goodmorning.goodbye.hello.com", false},
		{"end wildcard allowed", "http://tomorrow.eu", false},
		{"double end wildcard disallowed", "http://tomorrow.co.uk", true},
		{"athird.com disallowed", "http://athird.com", true},
		{"wildcard IP allowed", "http://100.11.12.13", false},
		{"IP disallowed", "http://101.12.13.14", true},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, test.shouldError)
	}
}

func TestBlockedDomains(t *testing.T) {
	rules := defaultHTTPRules()
	rules.BlockedDomains = []string{"example.com", "another.com", "*.hello.com", "tomorrow.*"}

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"example.com disallowed", "http://example.com", true},
		{"another.com disallowed", "http://another.com", true},
		{"wildcard disallowed", "http://goodbye.hello.com", true},
		{"double wildcard disallowed", "http://goodnight.goodbye.hello.com", true},
		{"end wildcard disallowed", "http://tomorrow.eu", true},
		{"double end wildcard allowed", "http://tomorrow.co.uk", false},
		{"athird.com allowed", "http://athird.com", false},
		{"IP allowed", "http://100.11.12.13", false},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, test.shouldError)
	}
}

func TestAllowedPorts(t *testing.T) {
	rules := defaultHTTPRules()
	rules.AllowedPorts = []int{8080}

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"standard http port allowed", "http://example.com", false},
		{"standard https port allowed", "https://example.com", false},
		{"port 8080 allowed", "http://example.com:8080", false},
		{"port 8088 disallowed", "http://example.com:8088", true},
		{},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, test.shouldError)
	}
}

func TestBlockedPorts(t *testing.T) {
	rules := defaultHTTPRules()
	rules.AllowedPorts = []int{8081, 8082}
	rules.BlockedPorts = []int{80, 443, 8080, 8081}

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"standard HTTP port disallowed", "http://example.com", true},
		{"standard HTTPS port disallowed", "https://example.com", true},
		{"port 8080 disallowed", "http://example.com", true},
		{"blocked list takes precedence over allow list", "http://example.com:8081", true},
		{"port 8082 allowed", "http://example.com:8082", false},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, test.shouldError)
	}
}

func TestBlockedWithCNAME(t *testing.T) {
	rules := defaultHTTPRules()
	rules.BlockedDomains = []string{"suborbital.network"}

	tests := []struct {
		name string
		url  string
	}{
		{"Resolved CNAME blocked", "https://test.suborbital.dev"},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, true)
	}
}

func TestDisallowedIPs(t *testing.T) {
	rules := defaultHTTPRules()
	rules.AllowIPs = false

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"IP disallowed", "http://100.11.12.13", true},
		{"Localhost IP disallowed", "http://127.0.0.1", true},
		{"Private IP disallowed", "http://192.168.0.11", true},
		{"Loopback IPv6 disallowed", "http://[::1]", true},
		{"Localhost IPv6 disallowed", "http://[fe80::1]", true},
		{"Private IPv6 disallowed", "http://[fd00::2f00]", true},
		{"Public IPv6 disallowed", "http://[2604:a880:cad:d0::dff:7001]", true},
		{"domain allowed", "http://friday.com", false},
		{"localhost allowed", "http://localhost:8080", false},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, test.shouldError)
	}
}

func TestDisallowedLocal(t *testing.T) {
	rules := defaultHTTPRules()
	rules.AllowPrivate = false

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"Loopback IPv6 disallowed", "http://[::1]", true},
		{"Localhost IPv6 disallowed", "http://[fe80::1]", true},
		{"Private IPv6 disallowed", "http://[fd00::2f00]", true},
		{"Resolves to Private disallowed", "http://local.suborbital.network", true},
		{"Resolves to Private (with port) disallowed", "http://local.suborbital.network:8081", true},
		{"Private disallowed", "http://localhost", true},
		{"Localhost IP disallowed", "http://127.0.0.1", true},
		{"Private IP disallowed", "http://192.168.0.11", true},
		{"Resolves to public allowed", "https://suborbital.dev", false},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, test.shouldError)
	}
}

func TestDisallowHTTP(t *testing.T) {
	rules := defaultHTTPRules()
	rules.AllowHTTP = false

	tests := []struct {
		name        string
		url         string
		shouldError bool
	}{
		{"HTTP disallowed", "http://example.com", true},
		{"HTTPS allowed", "https://friday.com", false},
	}

	for _, test := range tests {
		testRequestIsAllowed(t, test.name, rules, test.url, test.shouldError)
	}
}
