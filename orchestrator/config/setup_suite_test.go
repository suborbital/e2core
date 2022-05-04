package config_test

import (
	"os"

	"github.com/stretchr/testify/suite"
)

// ConfigTestSuite tests parsing configuration values and error handling.
type ConfigTestSuite struct {
	suite.Suite
	execMode     string
	satTag       string
	atmoTag      string
	controlPlane string
	envToken     string
	upstreamHost string
}

// SetupSuite will save the values of the environment variables internally, so we can restore them at the end to what
// they were before the tests ran.
func (cts *ConfigTestSuite) SetupSuite() {
	cts.T().Helper()

	cts.execMode = os.Getenv("VELOCITY_EXEC_MODE")
	cts.satTag = os.Getenv("VELOCITY_SAT_VERSION")
	cts.controlPlane = os.Getenv("VELOCITY_CONTROL_PLANE")
	cts.envToken = os.Getenv("VELOCITY_ENV_TOKEN")
	cts.upstreamHost = os.Getenv("VELOCITY_UPSTREAM_HOST")
}

// TearDownSuite restores all the environment variables to what they were before the tests began.
func (cts *ConfigTestSuite) TearDownSuite() {
	cts.T().Helper()

	var err error

	err = os.Setenv("VELOCITY_EXEC_MODE", cts.execMode)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable VELOCITY_EXEC_MODE",
			err,
		)
	}

	err = os.Setenv("VELOCITY_SAT_VERSION", cts.satTag)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable VELOCITY_SAT_VERSION",
			err,
		)
	}

	err = os.Setenv("VELOCITY_CONTROL_PLANE", cts.controlPlane)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable VELOCITY_CONTROL_PLANE",
			err,
		)
	}

	err = os.Setenv("VELOCITY_ENV_TOKEN", cts.envToken)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable VELOCITY_ENV_TOKEN",
			err,
		)
	}

	err = os.Setenv("VELOCITY_UPSTREAM_HOST", cts.upstreamHost)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable VELOCITY_UPSTREAM_HOST",
			err,
		)
	}
}

// SetupTest sets every environment variable value to empty string before any of the tests run. This method is also
// called from every subtest in the test functions.
func (cts *ConfigTestSuite) SetupTest() {
	cts.T().Helper()

	var err error
	envVars := []string{
		"VELOCITY_EXEC_MODE",
		"VELOCITY_SAT_VERSION",
		"VELOCITY_CONTROL_PLANE",
		"VELOCITY_ENV_TOKEN",
		"VELOCITY_UPSTREAM_HOST",
	}

	for _, v := range envVars {
		err = os.Unsetenv(v)
		if err != nil {
			cts.Require().FailNowf(
				"ConfigTestSuite.SetupTest",
				"tried to unset environment variable [%s], got error [%s]",
				v,
				err,
			)
		}
	}
}
