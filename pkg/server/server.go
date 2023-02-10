package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"o-neko-catnip/pkg/config"
	"o-neko-catnip/pkg/deployment"
	"o-neko-catnip/pkg/logger"
	"o-neko-catnip/pkg/metrics"
	"o-neko-catnip/pkg/oneko"
	"o-neko-catnip/pkg/oneko/service"
	"os"
	"os/signal"
	"syscall"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TriggerServer struct {
	configuration *config.Config
	log           *zap.SugaredLogger
	oneko         *service.Service
	monitor       *deployment.DeploymentMonitor
	appVersion    string
}

func New(c *config.Config, context context.Context, appVersion string) *TriggerServer {
	return &TriggerServer{
		log:           logger.New("server"),
		oneko:         service.New(c, context),
		monitor:       deployment.New(),
		configuration: c,
		appVersion:    appVersion,
	}
}

func (s *TriggerServer) Start() {
	metrics.RegisterCommonMetrics(s.appVersion)

	if s.configuration.ONeko.Mode == config.PRODUCTION {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	mainHandler := gin.New()
	otherHandler := gin.New()

	// logging
	mainHandler.Use(ginzap.GinzapWithConfig(s.log.Desugar(), &ginzap.Config{
		TimeFormat: time.RFC3339,
		UTC:        true,
		SkipPaths:  []string{"/up", "/metrics"},
	}))
	mainHandler.Use(ginzap.RecoveryWithZap(s.log.Desugar(), true))
	mainHandler.Use(s.catnipHeaderHandler())
	otherHandler.Use(ginzap.Ginzap(s.log.Desugar(), time.RFC3339, true))
	otherHandler.Use(ginzap.RecoveryWithZap(s.log.Desugar(), true))
	otherHandler.Use(s.catnipHeaderHandler())

	// custom template functions
	mainHandler.SetFuncMap(template.FuncMap{
		"formatAsDate": formatAsDate,
	})

	mainHandler.LoadHTMLGlob("frontend/dist/*.html")
	mainHandler.Static("/assets/", "frontend/dist/assets/")
	mainHandler.StaticFile("/favicon.ico", "public/assets/favicon.ico")
	mainHandler.GET("/metrics", metrics.PrometheusHandler())
	mainHandler.GET("/up", s.upHandler)

	mainHandler.GET("/", s.handleGetRequestToCatnipHome)
	mainHandler.GET("/:projectId/:versionId", s.handleGetRequestToWakeupUrl)
	mainHandler.NoRoute(s.redirectToHomePage)

	apiHandler := mainHandler.Group("/api")
	apiHandler.GET("/status", s.handleStatusRequest)

	otherHandler.GET("/*any", s.handleGetRequestToProjectUrl)

	address := fmt.Sprintf(":%d", s.configuration.ONeko.Server.Port)

	mux := newMux(mainHandler, otherHandler, s.oneko)

	srv := &http.Server{
		Addr:    address,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	s.log.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown:", err)
	}
}

type templateParameters struct {
	Project oneko.Project
	Version oneko.ProjectVersion
	BaseUrl string
}

func (s *TriggerServer) handleGetRequestToCatnipHome(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", templateParameters{
		BaseUrl: s.configuration.ONeko.Api.BaseUrl,
	})
}

func (s *TriggerServer) handleGetRequestToWakeupUrl(c *gin.Context) {
	projectId := c.Param("projectId")
	versionId := c.Param("versionId")

	project, version, err := s.oneko.GetProjectAndVersionByIds(projectId, versionId)

	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"error":   err.Error(),
			"BaseUrl": s.configuration.ONeko.Api.BaseUrl,
		})
		return
	}

	if !version.IsDeployed() {
		err := s.oneko.TriggerDeployment(projectId, versionId, c)
		if err != nil {
			c.HTML(http.StatusBadRequest, "error.html", gin.H{
				"error":   err.Error(),
				"BaseUrl": s.configuration.ONeko.Api.BaseUrl,
			})
			return
		}
	}

	c.HTML(http.StatusOK, "wakeup.html", templateParameters{
		Project: *project,
		Version: *version,
		BaseUrl: s.configuration.ONeko.Api.BaseUrl,
	})
}

func (s *TriggerServer) handleGetRequestToProjectUrl(c *gin.Context) {
	s.log.Debugw("incoming request to non-default url", "host", c.Request.Host)
	project, version, err := s.oneko.GetProjectAndVersionForUrl(fmt.Sprintf("%s%s", c.Request.Host, c.Request.RequestURI))
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	s.log.Debugw("request to url of project version", "project", project.Name, "version", version.Name)
	redirectUrl := s.getRedirectUrl(project, version, c)
	s.log.Debugw("redirecting", "url", redirectUrl)
	c.Redirect(http.StatusTemporaryRedirect, redirectUrl)
}

func (s *TriggerServer) getRedirectUrl(project *oneko.Project, version *oneko.ProjectVersion, c *gin.Context) string {
	protocol := getProtocol(c) + "://"
	return fmt.Sprintf("%s%s/%s/%s?redirectTo=%s%s%s", protocol, s.configuration.ONeko.CatnipUrl, project.Uuid, version.Uuid, protocol, c.Request.Host, c.Request.URL.Path)
}

func (s *TriggerServer) redirectToHomePage(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (s *TriggerServer) handleStatusRequest(c *gin.Context) {
	deploymentUrl, exists := c.GetQuery("deploymentUrl")
	if !exists {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	project, version, err := s.oneko.GetProjectAndVersionForUrl(deploymentUrl)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if !version.IsDeployed() {
		err = s.oneko.TriggerDeployment(project.Uuid, version.Uuid, c)
		if err != nil {
			_ = c.AbortWithError(http.StatusBadRequest, err)
			return
		}
	}

	status, err := s.monitor.DeploymentStatus(deploymentUrl)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

func getProtocol(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	} else {
		return "http"
	}
}

func formatAsDate(t time.Time) string {
	return t.Format(time.RFC1123)
}

func (s *TriggerServer) catnipHeaderHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("oneko-catnip", s.appVersion)
	}
}

func (s *TriggerServer) upHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}
