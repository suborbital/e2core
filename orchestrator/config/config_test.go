package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/suborbital/velocity/orchestrator/config"
)

func (cts *ConfigTestSuite) TestParse() {
	bundlePath := "./bundle.wasm.zip"

	tests := []struct {
		name    string
		args    []string
		setEnvs map[string]string
		want    config.Config
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "parses config correctly with correct environment variable values",
			args: []string{bundlePath},
			setEnvs: map[string]string{
				"VELOCITY_EXEC_MODE":     "metal",
				"VELOCITY_SAT_VERSION":   "1.0.2",
				"VELOCITY_CONTROL_PLANE": "controlplane.com:16384",
				"VELOCITY_ENV_TOKEN":     "envtoken.isajwt.butnotreally",
				"VELOCITY_UPSTREAM_HOST": "192.168.1.33:9888",
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
			name:    "parses the config with defaults, everything unset",
			args:    []string{bundlePath},
			setEnvs: map[string]string{},
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
			name:    "parses the config with defaults, do not pass bundlepath, receive error",
			args:    []string{},
			setEnvs: map[string]string{},
			want:    config.Config{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		cts.Run(tt.name, func() {
			cts.SetupTest()
			var err error

			for k, v := range tt.setEnvs {
				err = os.Setenv(k, v)
				if err != nil {
					cts.FailNowf(
						"set environment variable",
						"tried to set [%s] to [%s], got error [%s]",
						k,
						v,
						err,
					)
				}
			}

			subTestT := cts.T()

			got, err := config.Parse(tt.args[0])

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
