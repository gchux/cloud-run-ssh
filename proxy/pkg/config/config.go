package config

import (
	mapset "github.com/deckarep/golang-set/v2"

	cfg "go.uber.org/config"
)

type (
	accessControl struct {
		AllowedProjects   []string `yaml:"allowed_projects"`
		AllowedIdentities []string `yaml:"allowed_identities"`
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
		*proxyConfig
		AccessControl *AccessControl
	}
)

func LoadYAML(configFile *string) (*ProxyConfig, error) {
	provider, err := cfg.NewYAML(cfg.File(*configFile))
	if err != nil {
		return nil, err
	}

	_config := proxyConfig{}
	provider.Get(cfg.Root).Populate(&_config)

	config := &ProxyConfig{
		proxyConfig: &_config,
		AccessControl: &AccessControl{
			AllowedProjects:   mapset.NewSet(_config.AccessControl.AllowedProjects...),
			AllowedIdentities: mapset.NewSet(_config.AccessControl.AllowedIdentities...),
			AllowedHosts:      mapset.NewSet(_config.AccessControl.AllowedHosts...),
		},
	}

	return config, nil
}

func New(projectID string) *ProxyConfig {
	config := &ProxyConfig{
		proxyConfig: &proxyConfig{
			ProjectID: projectID,
		},
	}
	return config
}
