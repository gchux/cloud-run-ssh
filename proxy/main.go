package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/alphadose/haxmap"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
)

type (
	RevisionToInstancesMap = *haxmap.Map[string, mapset.Set[string]]
	ServiceToRevisionsMap  = *haxmap.Map[string, RevisionToInstancesMap]
	RegionToServicesMap    = *haxmap.Map[string, ServiceToRevisionsMap]
	ProjectToRegionsMap    = *haxmap.Map[string, RegionToServicesMap]
)

var id = flag.String("id", "", "allowed UUID to be registered")

const (
	sshProxyServerNameTemplate = "%s.ssh-server.internal"

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

var allUUIDs = uuid.Nil.String()

var (
	revisionToInstancesMap RevisionToInstancesMap = haxmap.New[string, mapset.Set[string]]()
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

		tokenValidator, err := idtoken.NewValidator(context.Background())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		payload, err := tokenValidator.Validate(context.Background(), idToken, *id)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if audience := payload.Audience; audience != sshProxyServerName {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}
}

func getIngessRules(c *gin.Context) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/plain")
	c.Writer.WriteHeaderNow()
	revisionToInstancesMap.ForEach(
		func(revision string, instances mapset.Set[string]) bool {
			instances.Each(func(instance string) bool {
				fmt.Fprintln(c.Writer, instance)
				return false
			})
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
	c.Header(xServerlessSshClientId, *id)
}

func addInstance(c *gin.Context) {
	clientID := c.GetHeader(xServerlessSshClientId)

	if clientID == "" {
		c.Status(http.StatusBadRequest)
		c.Header(xServerlessSshServerId, *id)
		return
	}

	project := c.Param("service")
	region := c.Param("region")
	service := c.Param("service")
	revision := c.Param("revision")
	instance := c.Param("instance")

	entry := fmt.Sprintf("%s %s", clientID, instance)

	projectToRegionsMap.GetOrCompute(project,
		func() RegionToServicesMap {
			regionToServicesMap.GetOrCompute(region,
				func() ServiceToRevisionsMap {
					serviceToRevisionsMap.GetOrCompute(service,
						func() RevisionToInstancesMap {
							if instances, loaded := revisionToInstancesMap.GetOrCompute(revision,
								func() mapset.Set[string] {
									return mapset.NewSet(entry)
								}); loaded {
								instances.Add(entry)
							}
							return revisionToInstancesMap
						})
					return serviceToRevisionsMap
				})
			return regionToServicesMap
		})

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

	if x, ok := projectToRegionsMap.Get(project); ok {
		if y, ok := x.Get(region); ok {
			if z, ok := y.Get(service); ok {
				if instances, ok := z.Get(revision); ok {
					if instances.Contains(instance) {
						instances.Remove(instance)
						sendResponse(c, http.StatusAccepted,
							&project, &region, &service, &revision, &instance, &clientID)
						return
					}
				}
			}
		}
	}

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

	externalAPI.GET("/ingress", getIngessRules)

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
	externalRevisionAPI.POST(instanceAPI, addInstance)
	externalRevisionAPI.DELETE(instanceAPI, removeInstance)

	internalRevisionAPI := internalServiceAPI.Group(revisionAPI)
	internalRevisionAPI.GET("/", getRevisionIngress)
	internalRevisionAPI.POST(instanceAPI, addInstance)
	internalRevisionAPI.DELETE(instanceAPI, removeInstance)

	go internalAPI.Run(":8888")
	externalAPI.Run(":8080")
}
