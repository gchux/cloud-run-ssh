package config

import (
	"os"

	mapset "github.com/deckarep/golang-set/v2"
	yaml "github.com/goccy/go-yaml"
)

type (
	accessControl struct {
		AllowedProjects   []string `yaml:"allowed_projects"`
		AllowedIdentities []string `yamls:"allowed_identities"`
		AllowedHosts      []string `yaml:"allowed_hosts"`
	}

	proxyConfig struct {
		ID            string        `yaml:"id"`
		ProjectID     string        `yaml:"project_id"`
		AccessControl accessControl `yaml:"access_control"`
	}

	AccessControl struct {
		AllowedProjects   mapset.Set[string]
		AllowedIdentities mapset.Set[string]
		AllowedHosts      mapset.Set[string]
	}

	ProxyConfig struct {
		proxyConfig
		AccessControl *AccessControl
	}
)

func LoadYAML(configFile *string) (*ProxyConfig, error) {
	data, err := os.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	var cfg proxyConfig
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	config := &ProxyConfig{
		proxyConfig: cfg,
		AccessControl: &AccessControl{
			AllowedProjects:   mapset.NewThreadUnsafeSet(cfg.AccessControl.AllowedProjects...),
			AllowedIdentities: mapset.NewThreadUnsafeSet(cfg.AccessControl.AllowedIdentities...),
			AllowedHosts:      mapset.NewThreadUnsafeSet(cfg.AccessControl.AllowedHosts...),
		},
	}

	return config, nil
}

func New(projectID string) *ProxyConfig {
	config := &ProxyConfig{
		proxyConfig: proxyConfig{
			ProjectID: projectID,
		},
	}
	return config
}
