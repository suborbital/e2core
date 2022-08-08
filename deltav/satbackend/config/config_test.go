package config_test

import (
	"testing"

	"github.com/sethvargo/go-envconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/suborbital/deltav/deltav/satbackend/config"
)

func (cts *ConfigTestSuite) TestParse() {
	bundlePath := "./bundle.wasm.zip"

	tests := []struct {
		name       string
		bundlePath string
		setEnvs    map[string]string
		want       config.Config
		wantErr    assert.ErrorAssertionFunc
	}{
		{
			name:       "parses config correctly with correct environment variable values",
			bundlePath: bundlePath,
			setEnvs: map[string]string{
				"CONSTD_EXEC_MODE":     "metal",
				"CONSTD_SAT_VERSION":   "1.0.2",
				"CONSTD_CONTROL_PLANE": "controlplane.com:16384",
				"CONSTD_ENV_TOKEN":     "envtoken.isajwt.butnotreally",
				"CONSTD_UPSTREAM_HOST": "192.168.1.33:9888",
			},
			want: config.Config{
				BundlePath:   bundlePath,
				ExecMode:     "metal",
				SatTag:       "1.0.2",
				ControlPlane: "controlplane.com:16384",
				EnvToken:     "envtoken.isajwt.butnotreally",
				UpstreamHost: "192.168.1.33:9888",
			},
			wantErr: assert.NoError,
		},
		{
			name:       "parses the config with defaults, everything unset",
			bundlePath: bundlePath,
			setEnvs:    map[string]string{},
			want: config.Config{
				BundlePath:   bundlePath,
				ExecMode:     "docker",
				SatTag:       "latest",
				ControlPlane: config.DefaultControlPlane,
				EnvToken:     "",
				UpstreamHost: "",
			},
			wantErr: assert.NoError,
		},
		{
			name:       "parses the config with defaults, do not pass bundlepath, receive error",
			bundlePath: "",
			setEnvs:    map[string]string{},
			want:       config.Config{},
			wantErr:    assert.Error,
		},
	}
	for _, tt := range tests {
		cts.Run(tt.name, func() {
			var err error

			subTestT := cts.T()

			got, err := config.Parse(tt.bundlePath, envconfig.MapLookuper(tt.setEnvs))

			tt.wantErr(subTestT, err)
			cts.Equal(tt.want, got)
		})
	}
}

// TestConfigTestSuite is the func that will run when `go test ./...` command is called. This encapsulates the suite and
// runs each of its tests.
func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
