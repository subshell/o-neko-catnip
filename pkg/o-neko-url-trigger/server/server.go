package server

import (
	"context"
	"errors"
	"fmt"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"html/template"
	"log"
	"net/http"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/logger"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/oneko"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type TriggerServer struct {
	configuration *config.Config
	log           *zap.SugaredLogger
	oneko         *oneko.ONekoApi
	appVersion    string
}

func New(c *config.Config, appVersion string) *TriggerServer {
	return &TriggerServer{
		log:           logger.New("server"),
		oneko:         oneko.New(c),
		configuration: c,
		appVersion: appVersion,
	}
}

func (s *TriggerServer) Start() {
	if s.configuration.ONeko.Mode == config.PRODUCTION {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()

	// logging
	r.Use(ginzap.Ginzap(s.log.Desugar(), time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(s.log.Desugar(), true))

	// routes
	r.LoadHTMLGlob("public/*.gotmpl")
	r.Static("/static/", "public/static/")
	r.GET("/metrics", prometheusHandler())
	r.GET("/", s.handleGetRequests)
	r.GET("/:any", s.handleGetRequests)
	r.HEAD("/", s.handleHeadRequests)
	r.HEAD("/:any", s.handleHeadRequests)

	// custom template functions
	r.SetFuncMap(template.FuncMap{
		"formatAsDate": formatAsDate,
	})

	address := fmt.Sprintf(":%d", s.configuration.ONeko.Server.Port)
	srv := &http.Server{
		Addr:    address,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Fatal("listen: %s\n", err)
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
}

func (s *TriggerServer) handleHeadRequests(c *gin.Context) {
	c.Header("oneko-url-trigger", s.appVersion)
	c.Status(http.StatusOK)
}

func (s *TriggerServer) handleGetRequests(c *gin.Context) {
	project, version, err := s.oneko.HandleRequest(c.Request.Host, c.Request.RequestURI)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html.gotmpl", gin.H{
			"error": err.Error(),
		})
		return
	}
	c.Header("oneko-url-trigger", s.appVersion)
	c.HTML(http.StatusOK, "index.html.gotmpl", templateParameters{
		Project: *project,
		Version: *version,
	})
}

func formatAsDate(t time.Time) string {
	return t.Format(time.RFC1123)
}
