package buildinfo

import "testing"

func TestCommitPrefersTheBuildValue(t *testing.T) {
	t.Setenv("VERCEL_GIT_COMMIT_SHA", "from-vercel")
	old := commit
	t.Cleanup(func() { commit = old })

	commit = "from-ldflags"
	if got := Commit(); got != "from-ldflags" {
		t.Errorf("Commit() = %q, want the value set at build time", got)
	}
	commit = ""
	if got := Commit(); got != "from-vercel" {
		t.Errorf("Commit() = %q, want VERCEL_GIT_COMMIT_SHA", got)
	}
}

func TestCommitNeverEmpty(t *testing.T) {
	t.Setenv("VERCEL_GIT_COMMIT_SHA", "")
	if Commit() == "" {
		t.Error("Commit() is empty; want a commit or \"unknown\"")
	}
}
