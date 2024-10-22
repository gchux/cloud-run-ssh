package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/alphadose/haxmap"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	oauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
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

var id = flag.String("id", "", "allowed UUID to be registered")

const (
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
	projectID = os.Getenv("PROJECT_ID")
	allUUIDs  = uuid.Nil.String()

	reaperInterval = 60 * time.Second
	maxIdleTimeout = 15 * time.Minute
)

var (
	instanceToConfigMap    InstanceToConfigMap    = haxmap.New[string, *ServerlessInstance]()
	revisionToInstancesMap RevisionToInstancesMap = haxmap.New[string, InstanceToConfigMap]()
	serviceToRevisionsMap  ServiceToRevisionsMap  = haxmap.New[string, RevisionToInstancesMap]()
	regionToServicesMap    RegionToServicesMap    = haxmap.New[string, ServiceToRevisionsMap]()
	projectToRegionsMap    ProjectToRegionsMap    = haxmap.New[string, RegionToServicesMap]()
)

func idTokenVerifier(id *string) func(*gin.Context) {
	sshProxyServerName := fmt.Sprintf(sshProxyServerNameTemplate, *id)

	return func(c *gin.Context) {
		sshServerID := c.GetHeader(xServerlessSshServerId)

		if sshServerID != *id {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		idToken := c.GetHeader(xServerlessSshAuthorization)

		if idToken == "" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		ctx := c.Request.Context()

		credentials, err := google.FindDefaultCredentials(ctx)
		if err == nil {
			fmt.Printf("oauth2[2]: %s | %s | %+v\n", projectID, sshProxyServerName, credentials)

			oauth2Service, oauth2Err := oauth2.NewService(ctx,
				option.WithCredentials(credentials),
				option.WithQuotaProject(projectID),
				option.WithRequestReason(sshProxyServerName))

			if oauth2Err == nil {
				tokenInfoCall := oauth2Service.Tokeninfo()
				tokenInfoCall.IdToken(idToken)
				tokenInfo, tokenInfoErr := tokenInfoCall.Do()

				if tokenInfoErr == nil && tokenInfo.Audience == sshProxyServerName {
					return
				}

				fmt.Println("oauth2[2]:", tokenInfoErr.Error())
			} else {
				fmt.Println("oauth2[3]", oauth2Err.Error())
			}
		} else {
			fmt.Println("oauth2[4]", err.Error())
		}

		tokenValidator, err := idtoken.NewValidator(ctx)
		if err != nil {
			fmt.Println("idtoken[1]", err.Error())
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		_, err = tokenValidator.Validate(ctx, idToken, sshProxyServerName)
		if err != nil {
			fmt.Println("idtoken[2]", err.Error())
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
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
	fmt.Fprintln(c.Writer)
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

func addInstance(c *gin.Context) {
	clientID := c.GetHeader(xServerlessSshClientId)

	if clientID == "" {
		c.Status(http.StatusBadRequest)
		c.Header(xServerlessSshServerId, *id)
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
		Tunnel:   &clientID,
		LastPing: &ts,
	}

	instanceToConfigMap.Set(instance, config)

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

	sendResponse(c, http.StatusAccepted,
		&project, &region, &service, &revision, &instance, &clientID)
}

func removeInstance(c *gin.Context) {
	clientID := c.GetHeader(xServerlessSshClientId)

	if clientID == "" {
		c.Status(http.StatusBadRequest)
		c.Header(xServerlessSshServerId, *id)
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
						if clientID == *config.Tunnel {
							if cfg, ok := instances.GetAndDel(*config.ID); ok {
								instanceToConfigMap.Del(*cfg.ID)
								sendResponse(c, http.StatusAccepted,
									cfg.Project, cfg.Region, cfg.Service, cfg.Revision, cfg.ID, cfg.Tunnel)
								return

							}
						}
					}
				}
			}
		}
	}

	sendResponse(c, http.StatusNotFound,
		&project, &region, &service, &revision, &instance, &clientID)
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
					if config, ok := instances.Get(instance); ok {
						if cfg, ok := instanceToConfigMap.Get(*config.ID); ok {
							if json, err := json.Marshal(*cfg); err == nil {
								sendResponse(c, http.StatusOK,
									cfg.Project, cfg.Region, cfg.Service, cfg.Revision, cfg.ID, cfg.Tunnel)
								c.Writer.Write(json)
							}
							return
						}
					}
				}
			}
		}
	}

	clientID := ""
	sendResponse(c, http.StatusNotFound,
		&project, &region, &service, &revision, &instance, &clientID)
}

func getProjectIngress(c *gin.Context) {
	project := c.Param("project")

	if project == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	regions, ok := projectToRegionsMap.Get(project)
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	c.Status(http.StatusOK)
	if json, err := regions.MarshalJSON(); err == nil {
		c.Writer.Write(json)
	}
}

func getRegionIngress(c *gin.Context) {
	project := c.Param("project")
	region := c.Param("region")

	if project == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if regions, ok := projectToRegionsMap.Get(project); ok {
		if services, ok := regions.Get(region); ok {
			if data, err := services.MarshalJSON(); err == nil {
				c.Status(http.StatusOK)
				c.Writer.Write(data)
				return
			} else {
				c.Status(http.StatusInternalServerError)
				return
			}
		}
	}

	c.Status(http.StatusNotFound)
}

func getServiceIngress(c *gin.Context) {
	project := c.Param("project")
	region := c.Param("region")
	service := c.Param("service")

	if project == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if regions, ok := projectToRegionsMap.Get(project); ok {
		if services, ok := regions.Get(region); ok {
			if revisions, ok := services.Get(service); ok {
				if data, err := revisions.MarshalJSON(); err == nil {
					c.Status(http.StatusOK)
					c.Writer.Write(data)
					return
				} else {
					c.Status(http.StatusInternalServerError)
					return
				}
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

	if project == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	if regions, ok := projectToRegionsMap.Get(project); ok {
		if services, ok := regions.Get(region); ok {
			if revisions, ok := services.Get(service); ok {
				if instances, ok := revisions.Get(revision); ok {
					if data, err := instances.MarshalJSON(); err == nil {
						c.Status(http.StatusOK)
						c.Writer.Write(data)
						return
					} else {
						c.Status(http.StatusInternalServerError)
						return
					}
				}
			}
		}
	}

	c.Status(http.StatusNotFound)
}

func main() {
	flag.Parse()

	if *id == "" || *id == allUUIDs {
		*id = allUUIDs
	} else if _id, err := uuid.Parse(*id); err != nil {
		*id = _id.String()
	} else {
		log.Fatalf("invalid id: %v", err)
	}

	fmt.Printf("use id '%s' to register instances\n", *id)

	gin.DisableConsoleColor()

	externalAPI := gin.Default()
	internalAPI := gin.Default()

	externalAPI.SetTrustedProxies(nil)
	internalAPI.SetTrustedProxies(nil)

	internalAPI.GET("/ingress", getIngessRules)

	externalProjectAPI := externalAPI.Group(projectAPI)
	internalProjectAPI := internalAPI.Group(projectAPI)

	externalProjectAPI.Use(idTokenVerifier(id))

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

	go idleInstancesReaper(&reaperInterval, &maxIdleTimeout)
	go internalAPI.Run(":8888")
	externalAPI.Run(":8080")
}
