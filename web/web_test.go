package web

import (
	"regexp"
	"strings"
	"testing"
)

// externalResource matches a script or stylesheet loaded from another host.
var externalResource = regexp.MustCompile(`<(?:script|link)\b[^>]*\b(?:src|href)="https?://[^"]*"[^>]*>`)

// Whatever the pages load from another host is pinned with subresource
// integrity, so a changed file on the CDN is refused instead of run.
func TestExternalResourcesArePinned(t *testing.T) {
	found := 0
	for _, name := range []string{"index.html", "docs.html"} {
		b, err := FS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, tag := range externalResource.FindAllString(string(b), -1) {
			found++
			if !strings.Contains(tag, `integrity="sha384-`) || !strings.Contains(tag, `crossorigin="anonymous"`) {
				t.Errorf("%s loads %s without subresource integrity", name, tag)
			}
		}
	}
	if found == 0 {
		t.Error("found no external resource; the pattern no longer matches docs.html")
	}
}
