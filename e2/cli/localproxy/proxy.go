package localproxy

import (
	"fmt"
	"io"
	"net/http"
)

// Proxy is a proxy from the local machine to the cloud-hosted editor.
type Proxy struct {
	endpoint string
	server   http.Server
	client   *http.Client
}

// New creates a new local proxy.
func New(endpoint string, listenPort string) *Proxy {
	p := &Proxy{
		endpoint: endpoint,
		client:   &http.Client{},
	}

	server := http.Server{
		Addr:    ":" + listenPort,
		Handler: p,
	}

	p.server = server

	return p
}

// Start starts the local proxy server.
func (p *Proxy) Start() error {
	fmt.Println("\nPROXY: local tunnel to function editor starting")

	return p.server.ListenAndServe()
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proxiedReq := *r
	proxiedReq.RequestURI = ""
	proxiedReq.Host = p.endpoint
	proxiedReq.URL.Host = p.endpoint
	proxiedReq.URL.Scheme = "https"

	resp, err := p.client.Do(&proxiedReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	header := w.Header()
	for k, vals := range resp.Header {
		for _, v := range vals {
			header.Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}
