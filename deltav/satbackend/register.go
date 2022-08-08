package satbackend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/suborbital/deltav/deltav/satbackend/config"
)

// AddUpstreamRequest is a request to add an upstream
type AddUpstreamRequest struct {
	UpstreamAddress string `json:"upstreamAddress"`
}

func registerWithControlPlane(conf config.Config) error {
	if conf.ControlPlane == config.DefaultControlPlane {
		return nil
	}

	var selfIPs []net.IP
	if conf.UpstreamHost != "" {
		selfIPs = []net.IP{net.ParseIP(conf.UpstreamHost)}
	} else {
		detectedIPs, err := getSelfIPAddress()
		if err != nil {
			return errors.Wrap(err, "failed to getSelfIPAddress")
		}

		selfIPs = detectedIPs
	}

	registerURL := fmt.Sprintf("%s/api/v1/upstream/register", conf.ControlPlane)

	for _, ip := range selfIPs {
		upstreamURL, err := url.Parse(fmt.Sprintf("http://%s:%s", ip.String(), atmoPort))
		if err != nil {
			return errors.Wrap(err, "failed to Parse")
		}

		payload := &AddUpstreamRequest{
			UpstreamAddress: upstreamURL.Host,
		}

		bodyJSON, err := json.Marshal(payload)
		if err != nil {
			return errors.Wrap(err, "failed to Marshal")
		}

		req, err := http.NewRequest(http.MethodPost, registerURL, bytes.NewBuffer(bodyJSON))
		if err != nil {
			return errors.Wrap(err, "failed to NewRequest")
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "failed to Do request")
		}

		if resp.StatusCode != http.StatusCreated {
			return errors.New("registration request failed: " + resp.Status)
		}
	}

	return nil
}

func getSelfIPAddress() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "failed to Interfaces")
	}

	ips := []net.IP{}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return nil, errors.Wrap(err, "failed to Addrs")
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if !ip.IsLoopback() && ip.IsPrivate() && ip.To4() != nil {
				ips = append(ips, ip)
			}
		}
	}

	return ips, nil
}
