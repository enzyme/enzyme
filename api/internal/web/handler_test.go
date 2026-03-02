package web

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestBuildIndexHTML(t *testing.T) {
	const html = `<!doctype html><html><head><title>Enzyme</title></head><body></body></html>`

	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(html)},
	}

	tests := []struct {
		name             string
		telemetryEnabled bool
		contains         string
		absent           string
	}{
		{
			name:             "telemetry disabled returns unmodified HTML",
			telemetryEnabled: false,
			absent:           "__ENZYME_CONFIG__",
		},
		{
			name:             "telemetry enabled injects config script",
			telemetryEnabled: true,
			contains:         `<script>window.__ENZYME_CONFIG__={"telemetry":true}</script></head>`,
		},
		{
			name:             "telemetry enabled preserves rest of HTML",
			telemetryEnabled: true,
			contains:         "<body></body>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(buildIndexHTML(fsys, tt.telemetryEnabled))
			if tt.contains != "" && !strings.Contains(result, tt.contains) {
				t.Errorf("expected HTML to contain %q\ngot: %s", tt.contains, result)
			}
			if tt.absent != "" && strings.Contains(result, tt.absent) {
				t.Errorf("expected HTML to NOT contain %q\ngot: %s", tt.absent, result)
			}
		})
	}
}
