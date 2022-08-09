package oneko

import (
	"context"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"o-neko-catnip/pkg/o-neko-catnip/config"
	"os"
	"testing"
	"time"
)

var (
	uut *ONekoApi
)

func TestMain(m *testing.M) {
	setTestConfiguration()
	uut = New(config.Configuration(), context.Background())
	os.Exit(m.Run())
}

func beforeEach() {
	httpmock.ActivateNonDefault(uut.client.GetClient())
}

func afterEach(t *testing.T) {
	httpmock.DeactivateAndReset()
	uut.cache.DeleteAll()
}

func setTestConfiguration() {
	config.OverrideConfiguration(&config.Config{
		ONeko: config.ONekoConfig{
			Api: config.ApiConfig{
				BaseUrl: "https://oneko.com",
				Auth: config.AuthConfig{
					Username: "admin",
					Password: "s3cr3t",
				},
				CacheRequestsInMinutes: 5,
			},
			Mode: "production",
			Server: config.ServerConfig{
				Port: 8080,
			},
			Logging: config.LoggingConfig{
				Level: "debug",
			},
		},
	})
}

func Test_PingCallsCorrectAPIEndpoint(t *testing.T) {
	beforeEach()
	defer afterEach(t)

	pingResponse := httpmock.NewStringResponder(200, "")
	httpmock.RegisterResponder("GET", "https://oneko.com/api/session", pingResponse)

	err := uut.ping()
	assert.NoError(t, err)
}

func Test_PingReturnsError(t *testing.T) {
	beforeEach()
	defer afterEach(t)

	pingResponse := httpmock.NewStringResponder(401, "")
	httpmock.RegisterResponder("GET", "https://oneko.com/api/session", pingResponse)

	err := uut.ping()
	assert.Error(t, err)
}

func Test_HandleRequest(t *testing.T) {
	beforeEach()
	defer afterEach(t)
	setupWakeupResponse(t)

	host := "https://my-test-instance.oneko.company.cloud"
	uri := "/foo/index.html"

	project, version, err := uut.HandleRequest(host, uri)

	assert.NoError(t, err)
	assert.NotNil(t, project)
	assert.NotNil(t, version)

	assert.Equal(t, "Demo Project", project.Name)
	assert.Equal(t, "demoversion-for-unittest", version.Name)
}

func Test_HandleRequest_ServeResponseFromCache(t *testing.T) {
	beforeEach()
	defer afterEach(t)
	setupWakeupResponse(t)

	host := "https://my-test-instance.oneko.company.cloud"
	uri := "/foo/index.html"

	testRequest := func() {
		project, version, err := uut.HandleRequest(host, uri)
		assert.NoError(t, err)
		assert.Equal(t, "Demo Project", project.Name)
		assert.Equal(t, "demoversion-for-unittest", version.Name)
	}

	testRequest()
	testRequest()

	callCount := httpmock.GetCallCountInfo()["POST https://oneko.com/api/project/deploy/url"]
	assert.Equal(t, 1, callCount)
}

func Test_HandleRequest_NotFound(t *testing.T) {
	beforeEach()
	defer afterEach(t)

	httpmock.RegisterResponder("POST", "https://oneko.com/api/project/deploy/url", httpmock.NewStringResponder(404, ""))

	host := "https://my-test-instance.oneko.company.cloud"
	uri := "/foo/index.html"

	project, version, err := uut.HandleRequest(host, uri)

	assert.EqualError(t, err, "no version matching this url found")
	assert.Nil(t, project)
	assert.Nil(t, version)
}

func setupWakeupResponse(t *testing.T) {
	responder, err := httpmock.NewJsonResponder(200, &Project{
		Uuid:      "not-really-a-uuid",
		Name:      "Demo Project",
		ImageName: "docker.mycompany.com/demoproject",
		Versions: []ProjectVersion{
			{
				Uuid:             "also-not-a-uuid",
				Name:             "demoversion-for-unittest",
				Urls:             []string {
					"https://my-test-instance.oneko.company.cloud",
				},
				ImageUpdatedDate: time.Now(),
			},
		},
	})
	assert.NoError(t, err)
	httpmock.RegisterResponder("POST", "https://oneko.com/api/project/deploy/url", responder)
}
