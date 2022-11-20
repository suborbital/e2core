//go:build !docker

package main

import (
	"context"
	"time"

	"github.com/suborbital/e2core/e2/cli/release"
	"github.com/suborbital/e2core/e2/cli/util"
)

const checkVersionTimeout = 500 * time.Millisecond

func checkForUpdates() {
	ctx, cancel := context.WithTimeout(context.Background(), checkVersionTimeout)
	defer cancel()

	versionChan := make(chan string)

	go func() {
		msg := ""
		if version, err := release.CheckForLatestVersion(ctx); err != nil {
			msg = err.Error()
		} else if version != "" {
			msg = version
		}

		versionChan <- msg
	}()

	select {
	case msg := <-versionChan:
		if msg != "" {
			util.LogInfo(msg)
		}
	}
}
