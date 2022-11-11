package options

import (
	"testing"

	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name     string
		configs  map[string]string
		want     Options
		wantErr  assert.ErrorAssertionFunc
		wantUUID func(string) bool
		wantPort func(string) bool
	}{
		{
			name: "options gets assembled with everything set",
			configs: map[string]string{
				"SAT_ENV_TOKEN":                 "envtoken",
				"SAT_HTTP_PORT":                 "1234",
				"SAT_UUID":                      "63147f8b-cd25-4eba-acc2-6ff48e6970b6",
				"SAT_CONTROL_PLANE":             "https://localhost:9091",
				"SAT_TRACER_TYPE":               "custom1",
				"SAT_RUNNABLE_IDENT":            "ident52",
				"SAT_RUNNABLE_VERSION":          "v9.5.4",
				"SAT_TRACER_SERVICENAME":        "service 543",
				"SAT_TRACER_PROBABILITY":        "0.2254332",
				"SAT_TRACER_COLLECTOR_ENDPOINT": "localhost:4325",
				"SAT_TRACER_HONEYCOMB_ENDPOINT": "api.honeycomb.io:443",
				"SAT_TRACER_HONEYCOMB_APIKEY":   "hcapikey",
				"SAT_TRACER_HONEYCOMB_DATASET":  "hcdataset",
				"SAT_METRICS_TYPE":              "otel",
				"SAT_METRICS_SERVICENAME":       "metricsservice",
				"SAT_METRICS_OTEL_ENDPOINT":     "localhost:1111",
			},
			want: Options{
				EnvToken:     "envtoken",
				Port:         "1234",
				ProcUUID:     "63147f8b-cd25-4eba-acc2-6ff48e6970b6",
				ControlPlane: &ControlPlane{Address: "https://localhost:9091"},
				Ident:        &Ident{Data: "ident52"},
				Version:      &Version{Data: "v9.5.4"},
				TracerConfig: TracerConfig{
					TracerType:  "custom1",
					ServiceName: "service 543",
					Probability: 0.2254332,
					Collector: &CollectorConfig{
						Endpoint: "localhost:4325",
					},
					HoneycombConfig: &HoneycombConfig{
						Endpoint: "api.honeycomb.io:443",
						APIKey:   "hcapikey",
						Dataset:  "hcdataset",
					},
				},
				MetricsConfig: MetricsConfig{
					Type:        "otel",
					ServiceName: "metricsservice",
					OtelMetrics: &OtelMetricsConfig{Endpoint: "localhost:1111"},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "options get assembled with only minimal set",
			configs: map[string]string{
				"SAT_ENV_TOKEN": "envtoken 66",
				"SAT_HTTP_PORT": "12345",
				"SAT_UUID":      "63147f8b-cd25-4eba-acc2-6ff48e6970b6",
			},
			want: Options{
				EnvToken: "envtoken 66",
				Port:     "12345",
				ProcUUID: "63147f8b-cd25-4eba-acc2-6ff48e6970b6",
				TracerConfig: TracerConfig{
					TracerType:  "none",
					ServiceName: "sat",
					Probability: 0.5,
				},
				MetricsConfig: MetricsConfig{
					Type:        "none",
					ServiceName: "sat",
					OtelMetrics: nil,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "errors out on not-a-uuid",
			configs: map[string]string{
				"SAT_UUID": "not a uuid",
			},
			want:    Options{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(envconfig.MapLookuper(tt.configs))

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
