// Package buildinfo tells which Git commit the running build was made from.
package buildinfo

import (
	"os"
	"runtime/debug"
)

// commit can be set at build time, as the Dockerfile does:
//
//	go build -ldflags "-X github.com/ruanpato/polite-greetings/internal/buildinfo.commit=<sha>"
var commit string

// Commit returns the Git commit of the running build, or "unknown". It looks,
// in order, at the value set at build time, at VERCEL_GIT_COMMIT_SHA (which
// Vercel sets on its deployments) and at the version-control data Go embeds
// when it builds inside a Git checkout.
func Commit() string {
	if commit != "" {
		return commit
	}
	if c := os.Getenv("VERCEL_GIT_COMMIT_SHA"); c != "" {
		return c
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				return s.Value
			}
		}
	}
	return "unknown"
}
