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
	endpointMask  = "%s/system/v1/module/%s/%s/%s/%s"
	clientTimeout = 3 * time.Second
)

type SE2Config struct {
	Endpoint string
}

type SE2 struct {
	client   *http.Client
	logger   zerolog.Logger
	endpoint string
}

func NewSE2(config SE2Config, l zerolog.Logger) SE2 {
	return SE2{
		client:   &http.Client{Timeout: clientTimeout},
		logger:   l.With().Str("component", "se2-modfetcher").Logger(),
		endpoint: config.Endpoint,
	}
}

func (s SE2) Get(ctx context.Context, fqmn fqmn.FQMN) (tenant.WasmModuleRef, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(endpointMask, s.endpoint, fqmn.Tenant, fqmn.Ref, fqmn.Namespace, fqmn.Name), nil)
	if err != nil {
		return tenant.WasmModuleRef{}, errors.Wrap(err, "http.NewRequestWithContext")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return tenant.WasmModuleRef{}, errors.Wrap(err, "s.client.Do")
	}

	defer resp.Body.Close()

	var r tenant.Module

	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return tenant.WasmModuleRef{}, errors.Wrap(err, "json.NewDecoder(resp.Body).Decode")
	}

	return *r.WasmRef, nil
}

var _ worker.ModSource = &SE2{}
