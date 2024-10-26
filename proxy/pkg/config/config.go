package config

import (
	mapset "github.com/deckarep/golang-set/v2"

	cfg "go.uber.org/config"
)

type (
	accessControl struct {
		AllowedProjects   []string `yaml:"allowed_projects"`
		AllowedRegions    []string `yaml:"allowed_regions"`
		AllowedServices   []string `yaml:"allowed_services"`
		AllowedRevisions  []string `yaml:"allowed_revisions"`
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
		AllowedRegions    mapset.Set[string]
		AllowedServices   mapset.Set[string]
		AllowedRevisions  mapset.Set[string]
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
			AllowedIdentities: mapset.NewSet(_config.AccessControl.AllowedIdentities...),
			AllowedHosts:      mapset.NewSet(_config.AccessControl.AllowedHosts...),
		},
	}

	if len(_config.AccessControl.AllowedProjects) > 0 {
		config.AccessControl.AllowedProjects = mapset.NewSet(_config.AccessControl.AllowedProjects...)
	} else {
		config.AccessControl.AllowedProjects = mapset.NewSetWithSize[string](0)
	}

	if len(_config.AccessControl.AllowedRegions) > 0 {
		config.AccessControl.AllowedRegions = mapset.NewSet(_config.AccessControl.AllowedRegions...)
	} else {
		config.AccessControl.AllowedRegions = mapset.NewSetWithSize[string](0)
	}

	if len(_config.AccessControl.AllowedServices) > 0 {
		config.AccessControl.AllowedServices = mapset.NewSet(_config.AccessControl.AllowedServices...)
	} else {
		config.AccessControl.AllowedServices = mapset.NewSetWithSize[string](0)
	}

	if len(_config.AccessControl.AllowedRevisions) > 0 {
		config.AccessControl.AllowedRevisions = mapset.NewSet(_config.AccessControl.AllowedRevisions...)
	} else {
		config.AccessControl.AllowedRevisions = mapset.NewSetWithSize[string](0)
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
