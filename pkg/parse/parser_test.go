package parse

import (
	"strings"
	"testing"
)

func TestParseConfiguration(t *testing.T) {
	gc := ParseConfiguration("testdata/example.dapp.yaml")
	c := (*gc)["test"]

	if c.Image != "stterm" {
		t.Fatal("unexpected image:", c.Image)
	}

	expectedContent := `
FROM alpine
RUN apk add st
ENTRYPOINT [ "/usr/bin/st" ]`

	if strings.TrimSpace(c.Dockerfile) != strings.TrimSpace(expectedContent) {
		t.Fatal("unexpected content:\n" + c.Dockerfile)
	}
}
