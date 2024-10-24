package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/alphadose/haxmap"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"

	cfg "github.com/gchux/cloud-run-ssh/proxy/pkg/config"
)

type (
	ServerlessInstance struct {
		Project  *string    `json:"project"`
		Region   *string    `json:"region"`
		Service  *string    `json:"service"`
		Revision *string    `json:"revision"`
		ID       *string    `json:"instance"`
		Tunnel   *string    `json:"tunnel"`
		LastPing *time.Time `json:"ping"`
	}

	InstanceToConfigMap    = *haxmap.Map[string, *ServerlessInstance]
	RevisionToInstancesMap = *haxmap.Map[string, InstanceToConfigMap]
	ServiceToRevisionsMap  = *haxmap.Map[string, RevisionToInstancesMap]
	RegionToServicesMap    = *haxmap.Map[string, ServiceToRevisionsMap]
	ProjectToRegionsMap    = *haxmap.Map[string, RegionToServicesMap]
)

const (
	configContextKey = "ssh_proxy_server_config"

	configFile = "/etc/ssh_proxy_server/config.yaml"

	sshProxyServerNameTemplate = "%s.ssh-proxy.internal"

	xProjectId            = "x-project-id"
	xServerlessRegion     = "x-s8s-region"
	xServerlessProjectId  = "x-s8s-project-id"
	xServerlessService    = "x-s8s-service"
	xServerlessRevision   = "x-s8s-revision"
	xServerlessInstanceId = "x-s8s-instance-id"

	xServerlessSshClientId      = "x-s8s-ssh-client-id"
	xServerlessSshServerId      = "x-s8s-ssh-server-id"
	xServerlessSshAuthorization = "x-s8s-ssh-authorization"

	projectAPI  = "/project/:project"
	regionAPI   = "/region/:region"
	serviceAPI  = "/service/:service"
	revisionAPI = "/revision/:revision"
	instanceAPI = "/instance/:instance"
)

var (
	projectID        = os.Getenv("PROJECT_ID")
	sshProxyServerID = os.Getenv("SSH_PROXY_SERVER_ID")

	allUUIDs = uuid.Nil.String()

	reaperInterval = 60 * time.Second
	maxIdleTimeout = 15 * time.Minute

	authorizedTokenIssuers = mapset.NewSet(
		"https://accounts.google.com",
	)
)

var (
	instanceToConfigMap    InstanceToConfigMap    = haxmap.New[string, *ServerlessInstance]()
	revisionToInstancesMap RevisionToInstancesMap = haxmap.New[string, InstanceToConfigMap]()
	serviceToRevisionsMap  ServiceToRevisionsMap  = haxmap.New[string, RevisionToInstancesMap]()
	regionToServicesMap    RegionToServicesMap    = haxmap.New[string, ServiceToRevisionsMap]()
	projectToRegionsMap    ProjectToRegionsMap    = haxmap.New[string, RegionToServicesMap]()
)

func idTokenVerifier(config *cfg.ProxyConfig) func(*gin.Context) {
	sshProxyServerName := fmt.Sprintf(sshProxyServerNameTemplate, config.ID)

	accessControl := config.AccessControl
	allowedIdentities := accessControl.AllowedIdentities

	return func(c *gin.Context) {
		_sshProxyServerID := c.GetHeader(xServerlessSshServerId)

		if _sshProxyServerID != config.ID {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		idToken := c.GetHeader(xServerlessSshAuthorization)

		if idToken == "" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		ctx := c.Request.Context()

		tokenValidator, err := idtoken.NewValidator(ctx)
		if err != nil {
			fmt.Println("idtoken[1]", err.Error())
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		payload, err := tokenValidator.Validate(ctx, idToken, sshProxyServerName)
		if err != nil {
			fmt.Println("idtoken[2]", err.Error())
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if !authorizedTokenIssuers.Contains(payload.Issuer) {
			fmt.Printf("idtoken[3]: rejected token issuer '%s'\n", payload.Issuer)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if sshProxyServerName != payload.Audience {
			fmt.Printf("idtoken[4]: rejected token audience '%s'\n", payload.Audience)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		claims := payload.Claims

		identity := ""

		if email, ok := claims["email"]; !ok ||
			!allowedIdentities.Contains(email.(string)) {
			fmt.Printf("idtoken[5]: rejected identity '%s'\n", email)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		} else {
			identity = email.(string)
		}

		if emailVerified, ok := claims["email_verified"]; !ok || !emailVerified.(bool) {
			fmt.Printf("idtoken[6]: rejected identity '%s' with email not verified\n", identity)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		fmt.Printf("allowed: '%s' into %s[%s]\n", identity, c.Request.Method, c.Request.URL.Path)

		c.Set(configContextKey, config)
	}
}

func projectVerifier(config *cfg.ProxyConfig) func(*gin.Context) {
	accessControl := config.AccessControl
	allowedProjects := accessControl.AllowedProjects

	return func(c *gin.Context) {
		project := c.Param("project")

		if project == "" {
			c.Status(http.StatusBadRequest)
			return
		}

		if allowedProjects.Contains(project) {
			return
		}

		fmt.Printf("rejected project: '%s'\n", project)
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

func idleInstancesReaper(interval, timeout *time.Duration) {
	ticker := time.NewTicker(*interval)

	for range ticker.C {
		var reapedInstances atomic.Uint32
		instanceToConfigMap.ForEach(
			func(_ string, config *ServerlessInstance) bool {
				idle := time.Since(*config.LastPing)
				if idle >= *timeout {
					if instances, ok := revisionToInstancesMap.Get(*config.Revision); ok {
						if cfg, ok := instances.GetAndDel(*config.ID); ok {
							instanceToConfigMap.Del(*cfg.ID)
							reapedInstances.Add(1)
							fmt.Printf("reaped instance: %s[%s] | idle: %v\n", *cfg.ID, *cfg.Tunnel, idle)
						}
					}
				}
				return true
			})
		fmt.Printf("reaped %d instances\n", reapedInstances.Load())
	}
}

func getIngessRules(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/plain")
	c.Writer.WriteHeaderNow()
	instanceToConfigMap.ForEach(
		func(_ string, config *ServerlessInstance) bool {
			fmt.Fprintf(c.Writer, "%s %s", *config.ID, *config.Tunnel)
			return true
		})
}

func sendResponse(
	c *gin.Context,
	status int,
	project, region, service, revision, instance, id *string,
) {
	c.Status(status)

	c.Header(xProjectId, *project)
	c.Header(xServerlessRegion, *region)
	c.Header(xServerlessService, *service)
	c.Header(xServerlessRevision, *revision)
	c.Header(xServerlessInstanceId, *instance)

	if *id != "" {
		c.Header(xServerlessSshClientId, *id)
	}
}

func getProxyConfig(c *gin.Context) *cfg.ProxyConfig {
	if config, ok := c.Get(configContextKey); ok {
		if config, ok := config.(*cfg.ProxyConfig); ok {
			return config
		}
	}
	return nil
}

func getSSHProxyClientID(
	c *gin.Context,
	config *cfg.ProxyConfig,
) (*string, bool) {
	clientID := c.GetHeader(xServerlessSshClientId)

	if clientID == "" {
		c.Header(xServerlessSshServerId, config.ID)
		c.AbortWithError(http.StatusBadRequest,
			errors.New("missing SSH_PROXY_CLIENT_ID"))
		return nil, false
	}

	return &clientID, true
}

func addInstance(c *gin.Context) {
	proxyConfig := getProxyConfig(c)

	clientID, ok := getSSHProxyClientID(c, proxyConfig)
	if !ok {
		return
	}

	project := c.Param("project")
	region := c.Param("region")
	service := c.Param("service")
	revision := c.Param("revision")
	instance := c.Param("instance")

	ts := time.Now()

	config := &ServerlessInstance{
		Project:  &project,
		Region:   &region,
		Service:  &service,
		Revision: &revision,
		ID:       &instance,
		Tunnel:   clientID,
		LastPing: &ts,
	}

	// a common instances bucket – which might be slow – is used
	// to speed up ingress rules generation for the `Tunnel manager`
	go instanceToConfigMap.Set(instance, config)

	// revisions get their own buckets of instances to speed up POST/DELETE operations:
	// there are many more instances than `project/region/service/revision` combinatoins;
	// a Run revision with too many instances cmoing and going should mostly hotspot its bucket.
	instanceToConfigMapProvider := func() InstanceToConfigMap {
		configMap := haxmap.New[string, *ServerlessInstance]()
		configMap.Set(instance, config)
		return configMap
	}

	revisionToInstancesMapProvider := func() RevisionToInstancesMap {
		if instances, loaded := revisionToInstancesMap.GetOrCompute(revision, instanceToConfigMapProvider); loaded {
			instances.Set(instance, config)
		}
		return revisionToInstancesMap
	}

	serviceToRevisionsMapProvider := func() ServiceToRevisionsMap {
		if revisions, loaded := serviceToRevisionsMap.GetOrCompute(service, revisionToInstancesMapProvider); loaded {
			if instances, loaded := revisions.GetOrCompute(revision, instanceToConfigMapProvider); loaded {
				instances.Set(instance, config)
			}
		}
		return serviceToRevisionsMap
	}

	regionToServicesMapProvider := func() RegionToServicesMap {
		if services, loaded := regionToServicesMap.GetOrCompute(region, serviceToRevisionsMapProvider); loaded {
			if revisions, loaded := services.GetOrCompute(service, revisionToInstancesMapProvider); loaded {
				if instances, loaded := revisions.GetOrCompute(revision, instanceToConfigMapProvider); loaded {
					instances.Set(instance, config)
				}
			}
		}
		return regionToServicesMap
	}

	if regions, loaded := projectToRegionsMap.GetOrCompute(project, regionToServicesMapProvider); loaded {
		if services, loaded := regions.GetOrCompute(region, serviceToRevisionsMapProvider); loaded {
			if revisions, loaded := services.GetOrCompute(service, revisionToInstancesMapProvider); loaded {
				if instances, loaded := revisions.GetOrCompute(revision, instanceToConfigMapProvider); loaded {
					instances.Set(instance, config)
				}
			}
		}
	}

	sendResponse(c, http.StatusAccepted, &project, &region, &service, &revision, &instance, clientID)
}

func removeInstance(c *gin.Context) {
	proxyConfig := getProxyConfig(c)

	clientID, ok := getSSHProxyClientID(c, proxyConfig)
	if !ok {
		return
	}

	project := c.Param("project")
	region := c.Param("region")
	service := c.Param("service")
	revision := c.Param("revision")
	instance := c.Param("instance")

	if regions, ok := projectToRegionsMap.Get(project); ok {
		if services, ok := regions.Get(region); ok {
			if revisions, ok := services.Get(service); ok {
				if instances, ok := revisions.Get(revision); ok {
					if config, ok := instances.Get(instance); ok {
						if *clientID == *config.Tunnel {
							if cfg, ok := instances.GetAndDel(*config.ID); ok {
								go instanceToConfigMap.Del(*cfg.ID)
								sendResponse(c, http.StatusAccepted,
									cfg.Project, cfg.Region, cfg.Service,
									cfg.Revision, cfg.ID, cfg.Tunnel)
								return
							}
						}
					}
				}
			}
		}
	}

	sendResponse(c, http.StatusNotFound,
		&project, &region, &service, &revision, &instance, clientID)
}

func getInstanceByID(c *gin.Context) {
	instance := c.Param("instance")

	if instance == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if cfg, ok := instanceToConfigMap.Get(instance); ok {

		if json, err := json.Marshal(*cfg); err == nil {
			sendResponse(c, http.StatusOK,
				cfg.Project, cfg.Region, cfg.Service,
				cfg.Revision, cfg.ID, cfg.Tunnel)
			c.Writer.Write(json)
		} else {
			sendResponse(c, http.StatusInternalServerError,
				cfg.Project, cfg.Region, cfg.Service,
				cfg.Revision, cfg.ID, cfg.Tunnel)
			fmt.Fprintln(c.Writer, err.Error())
		}

		return
	}

	c.Status(http.StatusNotFound)
	fmt.Fprintln(c.Writer, instance)
}

func getInstance(c *gin.Context) {
	project := c.Param("project")
	region := c.Param("region")
	service := c.Param("service")
	revision := c.Param("revision")
	instance := c.Param("instance")

	if regions, ok := projectToRegionsMap.Get(project); ok {
		if services, ok := regions.Get(region); ok {
			if revisions, ok := services.Get(service); ok {
				if instances, ok := revisions.Get(revision); ok {
					if _, ok := instances.Get(instance); ok {
						getInstanceByID(c)
						return
					}
				}
			}
		}
	}

	clientID := ""
	sendResponse(c, http.StatusNotFound,
		&project, &region, &service, &revision, &instance, &clientID)
}

func sendIngressResponse(c *gin.Context, instances []*ServerlessInstance) {
	var data []byte
	var err error

	if data, err = json.Marshal(instances); err != nil {
		c.Status(http.StatusInternalServerError)
		fmt.Fprintln(c.Writer, err.Error())
		return
	}

	c.Status(http.StatusOK)
	c.Writer.Write(data)
}

func getProjectIngress(c *gin.Context) {
	project := c.Param("project")

	if project == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if _, ok := projectToRegionsMap.Get(project); !ok {
		c.Status(http.StatusNotFound)
		return
	}

	instances := []*ServerlessInstance{}

	instanceToConfigMap.ForEach(
		func(_ string, instance *ServerlessInstance) bool {
			if *instance.Project == project {
				instances = append(instances, instance)
			}
			return true
		})

	sendIngressResponse(c, instances)
}

func getRegionIngress(c *gin.Context) {
	project := c.Param("project")
	region := c.Param("region")

	if project == "" || region == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if regions, ok := projectToRegionsMap.Get(project); ok {
		if _, ok := regions.Get(region); ok {

			instances := []*ServerlessInstance{}
			instanceToConfigMap.ForEach(
				func(_ string, instance *ServerlessInstance) bool {
					if *instance.Project == project &&
						*instance.Region == region {
						instances = append(instances, instance)
					}
					return true
				})

			sendIngressResponse(c, instances)
			return
		}
	}

	c.Status(http.StatusNotFound)
}

func getServiceIngress(c *gin.Context) {
	project := c.Param("project")
	region := c.Param("region")
	service := c.Param("service")

	if project == "" || region == "" || service == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if regions, ok := projectToRegionsMap.Get(project); ok {
		if services, ok := regions.Get(region); ok {
			if _, ok := services.Get(service); ok {

				instances := []*ServerlessInstance{}
				instanceToConfigMap.ForEach(
					func(_ string, instance *ServerlessInstance) bool {
						if *instance.Project == project &&
							*instance.Region == region &&
							*instance.Service == service {
							instances = append(instances, instance)
						}
						return true
					})

				sendIngressResponse(c, instances)
				return
			}
		}
	}

	c.Status(http.StatusNotFound)
}

func getRevisionIngress(c *gin.Context) {
	project := c.Param("project")
	region := c.Param("region")
	service := c.Param("service")
	revision := c.Param("revision")

	if project == "" || region == "" || service == "" || revision == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if regions, ok := projectToRegionsMap.Get(project); ok {
		if services, ok := regions.Get(region); ok {
			if revisions, ok := services.Get(service); ok {
				if instances, ok := revisions.Get(revision); ok {

					_instances := []*ServerlessInstance{}
					instances.ForEach(
						func(_ string, instance *ServerlessInstance) bool {
							_instances = append(_instances, instance)
							return true
						})

					sendIngressResponse(c, _instances)
					return
				}
			}
		}
	}

	c.Status(http.StatusNotFound)
}

func main() {
	var config *cfg.ProxyConfig
	configYAML := configFile
	if c, err := cfg.LoadYAML(&configYAML); err == nil {
		sshProxyServerID = c.ID
		if c.ProjectID == "" {
			c.ProjectID = projectID
		}
		config = c
	} else {
		fmt.Println(err.Error())
		config = cfg.New(projectID)
	}

	if sshProxyServerID == "" {
		sshProxyServerID = allUUIDs
	} else if id, err := uuid.Parse(sshProxyServerID); err == nil {
		sshProxyServerID = id.String()
	} else {
		sshProxyServerID = allUUIDs
	}
	config.ID = sshProxyServerID

	fmt.Printf("use id '%s' to register instances\n", config.ID)

	gin.DisableConsoleColor()

	externalAPI := gin.Default()
	externalAPI.SetTrustedProxies(nil)
	externalAPI.Use(idTokenVerifier(config))

	internalAPI := gin.Default()
	internalAPI.SetTrustedProxies(nil)
	internalAPI.GET("/ingress", getIngessRules)

	externalProjectAPI := externalAPI.Group(projectAPI)
	externalProjectAPI.Use(projectVerifier(config))

	internalProjectAPI := internalAPI.Group(projectAPI)

	externalProjectAPI.GET("/", getProjectIngress)
	internalProjectAPI.GET("/", getProjectIngress)

	externalRegionAPI := externalProjectAPI.Group(regionAPI)
	externalRegionAPI.GET("/", getRegionIngress)

	internalRegionAPI := internalProjectAPI.Group(regionAPI)
	internalRegionAPI.GET("/", getRegionIngress)

	externalServiceAPI := externalRegionAPI.Group(serviceAPI)
	externalServiceAPI.GET("/", getServiceIngress)

	internalServiceAPI := internalRegionAPI.Group(serviceAPI)
	internalServiceAPI.GET("/", getServiceIngress)

	externalRevisionAPI := externalServiceAPI.Group(revisionAPI)
	externalRevisionAPI.GET("/", getRevisionIngress)
	externalRevisionAPI.GET(instanceAPI, getInstance)
	externalRevisionAPI.POST(instanceAPI, addInstance)
	externalRevisionAPI.DELETE(instanceAPI, removeInstance)

	internalRevisionAPI := internalServiceAPI.Group(revisionAPI)
	internalRevisionAPI.GET("/", getRevisionIngress)
	internalRevisionAPI.GET(instanceAPI, getInstance)
	internalRevisionAPI.POST(instanceAPI, addInstance)
	internalRevisionAPI.DELETE(instanceAPI, removeInstance)

	externalAPI.GET(instanceAPI, getInstanceByID)
	internalAPI.GET(instanceAPI, getInstanceByID)

	go idleInstancesReaper(&reaperInterval, &maxIdleTimeout)

	go internalAPI.Run(":8888")
	externalAPI.Run(":8080")
}
