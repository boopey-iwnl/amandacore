package content

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzContentCompilerRejectsMalformedPackages(f *testing.F) {
	f.Add([]byte(`{"package_id":"fuzz","version":"0.0.1","schema_version":"1"}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, payload []byte) {
		dir := t.TempDir()
		manifestPath := filepath.Join(dir, "package.json")
		if err := os.WriteFile(manifestPath, payload, 0o644); err != nil {
			t.Fatalf("write fuzz package: %v", err)
		}
		_ = CompileContentPackage(manifestPath)
	})
}
