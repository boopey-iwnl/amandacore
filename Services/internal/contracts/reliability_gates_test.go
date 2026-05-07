package contracts_test

import (
	"path/filepath"
	"testing"
)

func TestReliabilitySecurityGateFilesExist(t *testing.T) {
	repoRoot := findRepoRoot(t)
	required := []string{
		filepath.Join(repoRoot, ".github", "workflows", "ci.yml"),
		filepath.Join(repoRoot, ".github", "workflows", "security-sanity.yml"),
		filepath.Join(repoRoot, ".github", "workflows", "package-smoke.yml"),
		filepath.Join(repoRoot, "Infra", "qa", "Scan-ForbiddenArtifacts.ps1"),
		filepath.Join(repoRoot, "Infra", "qa", "Smoke-Test.ps1"),
		filepath.Join(repoRoot, "Infra", "qa", "Validate-UiSmokeChecklist.ps1"),
		filepath.Join(repoRoot, "Docs", "Contracts", "replication-v1.json"),
		filepath.Join(repoRoot, "Content", "Schemas", "content-package-v1.schema.json"),
		filepath.Join(repoRoot, "Docs", "Runbooks", "ReliabilitySecurityLocalChecks.md"),
		filepath.Join(repoRoot, "Docs", "Runbooks", "ReleaseGateChecklist.md"),
		filepath.Join(repoRoot, "Docs", "Runbooks", "UiManualSmokeTest.md"),
		filepath.Join(repoRoot, "Docs", "Engineering", "PRValidationPolicy.md"),
		filepath.Join(repoRoot, "Docs", "QA", "AlphaReleaseChecklist.md"),
		filepath.Join(repoRoot, "Docs", "QA", "UiReleaseCandidateChecklist.md"),
	}

	for _, path := range required {
		if !fileExists(path) {
			t.Fatalf("required reliability/security gate file is missing: %s", path)
		}
	}
}
