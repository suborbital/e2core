package modfethcer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/suborbital/e2core/nuexecutor/worker"
	"github.com/suborbital/systemspec/fqmn"
	"github.com/suborbital/systemspec/tenant"
)

const (
	// host, tenantID, ref, namespace, name
	moduleEndpointMask = "%s/system/v1/module/%s/%s/%s/%s"
	tenantEndpointMask = "%s/system/v1/tenant/%s"
	clientTimeout      = 3 * time.Second
)

type SE2Config struct {
	Endpoint string
}

type SE2 struct {
	client   *http.Client
	logger   zerolog.Logger
	endpoint string
}

var _ worker.ModSource = &SE2{}

func NewSE2(config SE2Config, l zerolog.Logger) SE2 {
	return SE2{
		client:   &http.Client{Timeout: clientTimeout},
		logger:   l.With().Str("component", "se2-modfetcher").Logger(),
		endpoint: config.Endpoint,
	}
}

func (s SE2) Get(ctx context.Context, fqmn fqmn.FQMN) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(moduleEndpointMask, s.endpoint, fqmn.Tenant, fqmn.Ref, fqmn.Namespace, fqmn.Name), nil)
	if err != nil {
		return nil, errors.Wrap(err, "http.NewRequestWithContext")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "s.client.Do")
	}

	defer resp.Body.Close()

	var r tenant.Module

	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return nil, errors.Wrap(err, "json.NewDecoder(resp.Body).Decode")
	}

	return r.WasmRef.Data, nil
}
