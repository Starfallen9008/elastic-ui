package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type AuthMethod string

const (
	AuthBasic  AuthMethod = "basic"
	AuthAPIKey AuthMethod = "apikey"
)

type Profile struct {
	KibanaURL       string     `json:"kibanaUrl"`
	ESURL           string     `json:"esUrl"`
	AuthMethod      AuthMethod `json:"authMethod"`
	Username        string     `json:"username,omitempty"`
	APIKeyHint      string     `json:"apiKeyHint,omitempty"`
	SkipTLSVerify   bool       `json:"skipTlsVerify"`
	Space           string     `json:"space,omitempty"`
	AgentID         string     `json:"agentId,omitempty"`
	HostName        string     `json:"hostName,omitempty"`
	AgentPolicyID   string     `json:"agentPolicyId,omitempty"`
	PackagePolicyID string     `json:"packagePolicyId,omitempty"`
	PolicyName      string     `json:"policyName,omitempty"`
	BoundAt         string     `json:"boundAt,omitempty"`
}

func (p Profile) Valid() bool {
	return p.KibanaURL != "" && p.ESURL != ""
}

type Settings struct {
	PolicySetupComplete  bool    `json:"policySetupComplete"`
	Profile              Profile `json:"profile"`
	ShellIntegration     bool    `json:"shellIntegration"`
	Notifications        bool    `json:"notifications"`
	NotificationMode     string  `json:"notificationMode"`
	NotificationRevision int     `json:"notificationRevision,omitempty"`
	CloseToTray          bool    `json:"closeToTray"`
}

func Default() *Settings {
	return &Settings{
		Notifications:    true,
		NotificationMode: "native",
		CloseToTray:      true,
	}
}

func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "ElasticDefendConsole")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LogDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	logs := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logs, 0o700); err != nil {
		return "", err
	}
	return logs, nil
}

func Load() (*Settings, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return nil, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.NotificationMode == "" {
		s.NotificationMode = "native"
	}
	return &s, nil
}

func (s *Settings) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
