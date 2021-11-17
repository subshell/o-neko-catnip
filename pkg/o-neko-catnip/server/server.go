package server

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"o-neko-catnip/pkg/o-neko-catnip/config"
	"o-neko-catnip/pkg/o-neko-catnip/logger"
	"o-neko-catnip/pkg/o-neko-catnip/metrics"
	"o-neko-catnip/pkg/o-neko-catnip/oneko"
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
	oneko         *oneko.ONekoApi
	appVersion    string
}

func New(c *config.Config, context context.Context, appVersion string) *TriggerServer {
	return &TriggerServer{
		log:           logger.New("server"),
		oneko:         oneko.New(c, context),
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

	r := gin.New()

	// logging
	r.Use(ginzap.Ginzap(s.log.Desugar(), time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(s.log.Desugar(), true))

	// custom template functions
	r.SetFuncMap(template.FuncMap{
		"formatAsDate": formatAsDate,
	})

	// routes
	r.LoadHTMLGlob("public/*.gotmpl")
	r.Static("/static/", "public/static/")
	r.StaticFile("/favicon.ico", "public/static/favicon.ico")
	r.GET("/metrics", metrics.PrometheusHandler())


	// gin does not allow to use "/*any" because it would conflict with the routes above
	// so we have to use three routes to get what we want
	r.GET("/", s.handleGetRequests) // 1. root
	r.GET("/:any", s.handleGetRequests) // 2. any one-segment-path
	r.GET("/:any/*any", s.handleGetRequests) // 3. any multi-segment-path
	// same for HEAD requests
	r.HEAD("/", s.handleHeadRequests)
	r.HEAD("/:any", s.handleHeadRequests)
	r.HEAD("/:any/*any", s.handleHeadRequests)

	address := fmt.Sprintf(":%d", s.configuration.ONeko.Server.Port)
	srv := &http.Server{
		Addr:    address,
		Handler: r,
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

func (s *TriggerServer) handleHeadRequests(c *gin.Context) {
	c.Header("oneko-catnip", s.appVersion)
	c.Status(http.StatusOK)
}

func (s *TriggerServer) handleGetRequests(c *gin.Context) {
	project, version, err := s.oneko.HandleRequest(c.Request.Host, c.Request.RequestURI)
	c.Header("oneko-catnip", s.appVersion)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html.gotmpl", gin.H{
			"error":   err.Error(),
			"BaseUrl": s.configuration.ONeko.Api.BaseUrl,
		})
	} else {
		c.HTML(http.StatusOK, "index.html.gotmpl", templateParameters{
			Project: *project,
			Version: *version,
			BaseUrl: s.configuration.ONeko.Api.BaseUrl,
		})
	}
}

func formatAsDate(t time.Time) string {
	return t.Format(time.RFC1123)
}
