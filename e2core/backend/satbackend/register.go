package satbackend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/e2core/options"
)

// AddUpstreamRequest is a request to add an upstream
type AddUpstreamRequest struct {
	UpstreamAddress string `json:"upstreamAddress"`
}

func registerWithControlPlane(logger zerolog.Logger, opts options.Options) error {
	if opts.ControlPlane == options.DefaultControlPlane {
		return nil
	}

	var selfIPs []net.IP
	if opts.UpstreamAddress != "" {
		selfIPs = []net.IP{net.ParseIP(opts.UpstreamAddress)}
	} else {
		detectedIPs, err := getSelfIPAddress()
		if err != nil {
			return errors.Wrap(err, "failed to getSelfIPAddress")
		}

		selfIPs = detectedIPs
	}

	// golang's URL parsing does strange things if the original parsed string has no scheme, so we have to do some string manipulation
	registerURLString := fmt.Sprintf("%s/api/v1/upstream/register", opts.ControlPlane)
	if !strings.HasPrefix(registerURLString, "https") && !strings.HasPrefix(registerURLString, "http") {
		registerURLString = "http://" + registerURLString
	}

	registerURL, err := url.Parse(registerURLString)
	if err != nil {
		return errors.Wrapf(err, "failed to url.Parse %s", registerURLString)
	}

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

		req, err := http.NewRequest(http.MethodPost, registerURL.String(), bytes.NewBuffer(bodyJSON))
		if err != nil {
			return errors.Wrap(err, "failed to NewRequest")
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "failed to Do request")
		}

		if resp.StatusCode == http.StatusNotFound {
			logger.Info().Str("function", "registerWithControlPlane").Msg("control plane does not support backend registration")
			return nil
		} else if resp.StatusCode != http.StatusCreated {
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
