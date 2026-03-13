package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	toml "github.com/pelletier/go-toml/v2"
)

const (
	DefaultTimeout = 30 * time.Second
	LocalFileName  = "hawkings.toml"
	HomeFileName   = ".hawkings.toml"
)

type File struct {
	Version  int                `toml:"version" json:"version"`
	Profile  string             `toml:"profile" json:"profile,omitempty"`
	Profiles map[string]Profile `toml:"profiles" json:"profiles,omitempty"`
}

type Profile struct {
	Environment  string `toml:"environment" json:"environment,omitempty"`
	BaseURL      string `toml:"base_url" json:"base_url,omitempty"`
	XAPIKey      string `toml:"x_api_key" json:"x_api_key,omitempty"`
	APIKey       string `toml:"api_key" json:"api_key,omitempty"`
	PlatformUUID string `toml:"platform_uuid" json:"platform_uuid,omitempty"`
	PlatformName string `toml:"platform_name" json:"platform_name,omitempty"`
	Timeout      string `toml:"timeout" json:"timeout,omitempty"`
}

type LoadOptions struct {
	CWD             string
	HomeDir         string
	LocalConfigPath string
}

type Paths struct {
	Local       string `json:"local"`
	LocalFound  bool   `json:"local_found"`
	Global      string `json:"global"`
	GlobalFound bool   `json:"global_found"`
}

type Sources struct {
	Paths         Paths  `json:"paths"`
	ActiveProfile string `json:"active_profile"`
}

type LoadResult struct {
	Config File  `json:"config"`
	Paths  Paths `json:"paths"`
}

type Overrides struct {
	Profile      string
	BaseURL      string
	XAPIKey      string
	APIKey       string
	PlatformUUID string
	Timeout      time.Duration
}

type ResolvedConfig struct {
	ProfileName  string        `json:"profile"`
	Environment  string        `json:"environment,omitempty"`
	BaseURL      string        `json:"base_url"`
	XAPIKey      string        `json:"x_api_key"`
	PlatformUUID string        `json:"platform_uuid,omitempty"`
	PlatformName string        `json:"platform_name,omitempty"`
	Timeout      time.Duration `json:"timeout"`
	Sources      Sources       `json:"sources"`
}

type ResolvedConfigView struct {
	Profile      string             `json:"profile"`
	Environment  string             `json:"environment,omitempty"`
	BaseURL      string             `json:"base_url"`
	XAPIKey      string             `json:"x_api_key"`
	PlatformUUID string             `json:"platform_uuid,omitempty"`
	PlatformName string             `json:"platform_name,omitempty"`
	Timeout      string             `json:"timeout"`
	Sources      Sources            `json:"sources"`
	Profiles     map[string]Profile `json:"profiles,omitempty"`
}

func Load(opts LoadOptions) (LoadResult, error) {
	cwd := opts.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return LoadResult{}, fmt.Errorf("resolve cwd: %w", err)
		}
	}

	homeDir := opts.HomeDir
	if homeDir == "" {
		homeDir, _ = os.UserHomeDir()
	}

	localPath := opts.LocalConfigPath
	if localPath == "" {
		foundPath, found := findNearestConfig(cwd)
		if found {
			localPath = foundPath
		} else {
			localPath = filepath.Join(cwd, LocalFileName)
		}
	}

	globalPath := ""
	if homeDir != "" {
		globalPath = filepath.Join(homeDir, HomeFileName)
	}
	paths := Paths{
		Local:  localPath,
		Global: globalPath,
	}

	merged := File{
		Version:  1,
		Profiles: map[string]Profile{},
	}

	if exists(localPath) {
		paths.LocalFound = true
		localCfg, err := loadFile(localPath)
		if err != nil {
			return LoadResult{}, fmt.Errorf("load local config %s: %w", localPath, err)
		}
		merged = Merge(merged, localCfg)
	}

	if exists(globalPath) {
		paths.GlobalFound = true
		globalCfg, err := loadFile(globalPath)
		if err != nil {
			return LoadResult{}, fmt.Errorf("load global config %s: %w", globalPath, err)
		}
		merged = Merge(globalCfg, merged)
	}

	return LoadResult{
		Config: merged,
		Paths:  paths,
	}, nil
}

func Merge(base, overlay File) File {
	out := File{
		Version:  base.Version,
		Profile:  base.Profile,
		Profiles: map[string]Profile{},
	}

	if out.Version == 0 {
		out.Version = 1
	}

	for key, value := range base.Profiles {
		out.Profiles[key] = value
	}

	if overlay.Version != 0 {
		out.Version = overlay.Version
	}
	if overlay.Profile != "" {
		out.Profile = overlay.Profile
	}
	for key, value := range overlay.Profiles {
		out.Profiles[key] = mergeProfile(out.Profiles[key], value)
	}

	return out
}

func Resolve(result LoadResult, overrides Overrides) (ResolvedConfig, error) {
	profileName := strings.TrimSpace(firstNonEmpty(
		overrides.Profile,
		os.Getenv("HAWKINGS_CLI_PROFILE"),
		result.Config.Profile,
	))
	if profileName == "" {
		return ResolvedConfig{}, errors.New("no active profile configured; set --profile, HAWKINGS_CLI_PROFILE, hawkings.toml profile, or ~/.hawkings.toml profile")
	}

	profile, ok := result.Config.Profiles[profileName]
	if !ok {
		return ResolvedConfig{}, fmt.Errorf("profile %q not found in hawkings.toml or ~/.hawkings.toml", profileName)
	}

	timeout, err := parseTimeout(profile.Timeout)
	if err != nil {
		return ResolvedConfig{}, fmt.Errorf("invalid timeout for profile %q: %w", profileName, err)
	}

	if overrides.Timeout > 0 {
		timeout = overrides.Timeout
	}

	baseURL := strings.TrimSpace(firstNonEmpty(
		overrides.BaseURL,
		os.Getenv("HAWKINGS_CLI_BASE_URL"),
		profile.BaseURL,
		baseURLFromEnvironment(profile.Environment),
	))
	if err := validateString("base URL", baseURL); err != nil {
		return ResolvedConfig{}, err
	}
	baseURL, err = normalizeBaseURL(baseURL)
	if err != nil {
		return ResolvedConfig{}, err
	}

	xAPIKey := strings.TrimSpace(firstNonEmpty(
		overrides.XAPIKey,
		os.Getenv("HAWKINGS_CLI_X_API_KEY"),
		profile.XAPIKey,
	))
	apiKey := strings.TrimSpace(firstNonEmpty(
		overrides.APIKey,
		os.Getenv("HAWKINGS_CLI_API_KEY"),
		profile.APIKey,
	))
	platformUUID := strings.TrimSpace(firstNonEmpty(
		overrides.PlatformUUID,
		os.Getenv("HAWKINGS_CLI_PLATFORM_UUID"),
		profile.PlatformUUID,
	))

	if xAPIKey == "" {
		if err := validateString("API key", apiKey); err != nil {
			return ResolvedConfig{}, err
		}
		if err := validateString("platform UUID", platformUUID); err != nil {
			return ResolvedConfig{}, err
		}
		if apiKey == "" || platformUUID == "" {
			return ResolvedConfig{}, errors.New("profile must define x_api_key or api_key + platform_uuid")
		}
		xAPIKey = apiKey + "-" + platformUUID
	}

	if err := validateString("x-api-key", xAPIKey); err != nil {
		return ResolvedConfig{}, err
	}

	return ResolvedConfig{
		ProfileName:  profileName,
		Environment:  strings.TrimSpace(profile.Environment),
		BaseURL:      baseURL,
		XAPIKey:      xAPIKey,
		PlatformUUID: platformUUID,
		PlatformName: strings.TrimSpace(profile.PlatformName),
		Timeout:      timeout,
		Sources: Sources{
			Paths:         result.Paths,
			ActiveProfile: profileName,
		},
	}, nil
}

func (c ResolvedConfig) RedactedView(result LoadResult) ResolvedConfigView {
	profiles := map[string]Profile{}
	for key, value := range result.Config.Profiles {
		copyValue := value
		copyValue.XAPIKey = redactSecret(copyValue.XAPIKey)
		copyValue.APIKey = redactSecret(copyValue.APIKey)
		profiles[key] = copyValue
	}

	return ResolvedConfigView{
		Profile:      c.ProfileName,
		Environment:  c.Environment,
		BaseURL:      c.BaseURL,
		XAPIKey:      redactSecret(c.XAPIKey),
		PlatformUUID: c.PlatformUUID,
		PlatformName: c.PlatformName,
		Timeout:      c.Timeout.String(),
		Sources:      c.Sources,
		Profiles:     profiles,
	}
}

func loadFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}

	cfg := File{
		Version:  1,
		Profiles: map[string]Profile{},
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return File{}, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

func mergeProfile(base, overlay Profile) Profile {
	out := base
	if overlay.Environment != "" {
		out.Environment = overlay.Environment
	}
	if overlay.BaseURL != "" {
		out.BaseURL = overlay.BaseURL
	}
	if overlay.XAPIKey != "" {
		out.XAPIKey = overlay.XAPIKey
	}
	if overlay.APIKey != "" {
		out.APIKey = overlay.APIKey
	}
	if overlay.PlatformUUID != "" {
		out.PlatformUUID = overlay.PlatformUUID
	}
	if overlay.PlatformName != "" {
		out.PlatformName = overlay.PlatformName
	}
	if overlay.Timeout != "" {
		out.Timeout = overlay.Timeout
	}
	return out
}

func parseTimeout(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return DefaultTimeout, nil
	}
	return time.ParseDuration(value)
}

func baseURLFromEnvironment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "dev", "development":
		return "https://dev-data-api.hawkings.education/v1"
	case "pro", "prod", "production":
		return "https://data-api.hawkings.education/v1"
	default:
		return ""
	}
}

func normalizeBaseURL(value string) (string, error) {
	trimmed := strings.TrimSpace(strings.TrimRight(value, "/"))
	if trimmed == "" {
		return "", errors.New("base URL is required")
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		return "", fmt.Errorf("base URL %q must start with http:// or https://", trimmed)
	}
	if !strings.Contains(trimmed, "/v") {
		trimmed += "/v1"
	}
	return trimmed, nil
}

func validateString(name, value string) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s contains invalid UTF-8", name)
	}
	for _, r := range value {
		if r < 0x20 {
			return fmt.Errorf("%s contains control characters", name)
		}
	}
	return nil
}

func exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func findNearestConfig(cwd string) (string, bool) {
	dir := cwd
	for {
		candidate := filepath.Join(dir, LocalFileName)
		if exists(candidate) {
			return candidate, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func redactSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "********"
	}
	return secret[:6] + "…" + secret[len(secret)-4:]
}
