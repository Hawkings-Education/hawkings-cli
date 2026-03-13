package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMergeUsesLocalProfileSelectionOnTopOfGlobalProfiles(t *testing.T) {
	global := File{
		Version: 1,
		Profile: "prod",
		Profiles: map[string]Profile{
			"dev": {
				Environment: "dev",
				XAPIKey:     "global-dev-token",
			},
		},
	}
	local := File{
		Profile: "dev",
		Profiles: map[string]Profile{
			"dev": {
				PlatformName: "Local Hawkings",
			},
		},
	}

	merged := Merge(global, local)
	if merged.Profile != "dev" {
		t.Fatalf("expected local profile to win, got %q", merged.Profile)
	}
	if merged.Profiles["dev"].XAPIKey != "global-dev-token" {
		t.Fatalf("expected global x_api_key to remain available")
	}
	if merged.Profiles["dev"].PlatformName != "Local Hawkings" {
		t.Fatalf("expected local platform name override to apply")
	}
}

func TestResolveBuildsXAPIKeyFromAPIKeyAndPlatformUUID(t *testing.T) {
	result := LoadResult{
		Config: File{
			Version: 1,
			Profile: "dev",
			Profiles: map[string]Profile{
				"dev": {
					Environment:  "dev",
					APIKey:       "hk-123",
					PlatformUUID: "platform-uuid",
					Timeout:      "45s",
				},
			},
		},
		Paths: Paths{
			Local:  filepath.Join("/tmp", LocalFileName),
			Global: filepath.Join("/tmp", HomeFileName),
		},
	}

	resolved, err := Resolve(result, Overrides{})
	if err != nil {
		t.Fatalf("resolve returned error: %v", err)
	}
	if resolved.XAPIKey != "hk-123-platform-uuid" {
		t.Fatalf("unexpected x-api-key: %q", resolved.XAPIKey)
	}
	if resolved.BaseURL != "https://dev-data-api.hawkings.education/v1" {
		t.Fatalf("unexpected base URL: %q", resolved.BaseURL)
	}
	if resolved.Timeout != 45*time.Second {
		t.Fatalf("unexpected timeout: %s", resolved.Timeout)
	}
}

func TestFindNearestConfigWalksUpDirectories(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	configPath := filepath.Join(root, LocalFileName)
	if err := os.WriteFile(configPath, []byte("profile = \"dev\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	found, ok := findNearestConfig(nested)
	if !ok {
		t.Fatalf("expected config to be found")
	}
	if found != configPath {
		t.Fatalf("expected %q, got %q", configPath, found)
	}
}
