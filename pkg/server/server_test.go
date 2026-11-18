package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"o-neko-catnip/pkg/config"
	"o-neko-catnip/pkg/oneko"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetProtocol(t *testing.T) {
	testCases := []struct {
		name             string
		forwardedProto   string
		hasTLS           bool
		expectedProtocol string
	}{
		{
			name:             "default HTTP when no TLS and no header",
			forwardedProto:   "",
			hasTLS:           false,
			expectedProtocol: "http",
		},
		{
			name:             "HTTPS when TLS is present and no header",
			forwardedProto:   "",
			hasTLS:           true,
			expectedProtocol: "https",
		},
		{
			name:             "X-Forwarded-Proto http",
			forwardedProto:   "http",
			hasTLS:           false,
			expectedProtocol: "http",
		},
		{
			name:             "X-Forwarded-Proto https",
			forwardedProto:   "https",
			hasTLS:           false,
			expectedProtocol: "https",
		},
		{
			name:             "X-Forwarded-Proto uppercase HTTPS",
			forwardedProto:   "HTTPS",
			hasTLS:           false,
			expectedProtocol: "https",
		},
		{
			name:             "X-Forwarded-Proto with whitespace",
			forwardedProto:   "  https  ",
			hasTLS:           false,
			expectedProtocol: "https",
		},
		{
			name:             "X-Forwarded-Proto comma-separated list takes first entry (https)",
			forwardedProto:   "https, http",
			hasTLS:           false,
			expectedProtocol: "https",
		},
		{
			name:             "X-Forwarded-Proto comma-separated list takes first entry (http)",
			forwardedProto:   "http, https",
			hasTLS:           true,
			expectedProtocol: "http",
		},
		{
			name:             "X-Forwarded-Proto empty with TLS",
			forwardedProto:   "",
			hasTLS:           true,
			expectedProtocol: "https",
		},
		{
			name:             "X-Forwarded-Proto whitespace only with TLS falls back to TLS",
			forwardedProto:   "   ",
			hasTLS:           true,
			expectedProtocol: "https",
		},
		{
			name:             "X-Forwarded-Proto whitespace only without TLS falls back to http",
			forwardedProto:   "   ",
			hasTLS:           false,
			expectedProtocol: "http",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
			if testCase.forwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", testCase.forwardedProto)
			}
			if testCase.hasTLS {
				req.TLS = &tls.ConnectionState{}
			}
			ctx.Request = req

			actualProtocol := getProtocol(ctx)
			assert.Equal(t, testCase.expectedProtocol, actualProtocol)
		})
	}
}

func TestGetRedirectUrl(t *testing.T) {
	server := &TriggerServer{
		configuration: &config.Config{
			ONeko: config.ONekoConfig{
				CatnipUrl: "catnip.example.com",
			},
		},
	}

	project := &oneko.Project{
		Uuid: "proj-123",
	}
	version := &oneko.ProjectVersion{
		Uuid: "ver-456",
	}

	testCases := []struct {
		name           string
		forwardedProto string
		hasTLS         bool
		host           string
		path           string
		expectedURL    string
	}{
		{
			name:           "Standard HTTP redirect",
			forwardedProto: "",
			hasTLS:         false,
			host:           "app.example.com",
			path:           "/dashboard",
			expectedURL:    "http://catnip.example.com/wakeup?projectId=proj-123&versionId=ver-456&redirectTo=http://app.example.com/dashboard",
		},
		{
			name:           "HTTPS with X-Forwarded-Proto redirect",
			forwardedProto: "https",
			hasTLS:         false,
			host:           "app.example.com",
			path:           "/feature",
			expectedURL:    "https://catnip.example.com/wakeup?projectId=proj-123&versionId=ver-456&redirectTo=https://app.example.com/feature",
		},
		{
			name:           "Multi-proxy X-Forwarded-Proto header",
			forwardedProto: "https, http",
			hasTLS:         false,
			host:           "app.example.com",
			path:           "/login",
			expectedURL:    "https://catnip.example.com/wakeup?projectId=proj-123&versionId=ver-456&redirectTo=https://app.example.com/login",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			req := httptest.NewRequest(http.MethodGet, "http://"+testCase.host+testCase.path, nil)
			if testCase.forwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", testCase.forwardedProto)
			}
			if testCase.hasTLS {
				req.TLS = &tls.ConnectionState{}
			}
			ctx.Request = req

			redirectURL := server.getRedirectUrl(project, version, ctx)
			assert.Equal(t, testCase.expectedURL, redirectURL)
		})
	}
}
