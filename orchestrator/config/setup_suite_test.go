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

	cts.execMode = os.Getenv("CONSTD_EXEC_MODE")
	cts.satTag = os.Getenv("CONSTD_SAT_VERSION")
	cts.atmoTag = os.Getenv("CONSTD_ATMO_VERSION")
	cts.controlPlane = os.Getenv("CONSTD_CONTROL_PLANE")
	cts.envToken = os.Getenv("CONSTD_ENV_TOKEN")
	cts.upstreamHost = os.Getenv("CONSTD_UPSTREAM_HOST")
}

// TearDownSuite restores all the environment variables to what they were before the tests began.
func (cts *ConfigTestSuite) TearDownSuite() {
	cts.T().Helper()

	var err error

	err = os.Setenv("CONSTD_EXEC_MODE", cts.execMode)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable CONSTD_EXEC_MODE",
			err,
		)
	}

	err = os.Setenv("CONSTD_SAT_VERSION", cts.satTag)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable CONSTD_SAT_VERSION",
			err,
		)
	}

	err = os.Setenv("CONSTD_ATMO_VERSION", cts.atmoTag)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable CONSTD_ATMO_VERSION",
			err,
		)
	}

	err = os.Setenv("CONSTD_CONTROL_PLANE", cts.controlPlane)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable CONSTD_CONTROL_PLANE",
			err,
		)
	}

	err = os.Setenv("CONSTD_ENV_TOKEN", cts.envToken)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable CONSTD_ENV_TOKEN",
			err,
		)
	}

	err = os.Setenv("CONSTD_UPSTREAM_HOST", cts.upstreamHost)
	if err != nil {
		cts.Assert().FailNow("tear down failed",
			"can't restore environment variable CONSTD_UPSTREAM_HOST",
			err,
		)
	}
}
