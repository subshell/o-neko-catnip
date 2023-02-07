package api

import (
	"context"
	"fmt"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"net/http"
	"o-neko-catnip/pkg/config"
	"o-neko-catnip/pkg/oneko"
	"os"
	"testing"
	"time"
)

const (
	projectUuid string = "63638583-b9d0-4245-8610-e19c040e6e10"
	versionUuid string = "5eb9c99f-e1d8-4a70-b394-725de9b4ab0d"
	versionUrl  string = "my-test-instance.oneko.company.cloud"
)

var (
	uut         *Api
	demoProject = &oneko.Project{
		Uuid:      projectUuid,
		Name:      "Demo Project",
		ImageName: "docker.mycompany.com/demoproject",
		Versions: []oneko.ProjectVersion{
			{
				Uuid: versionUuid,
				Name: "demoversion-for-unittest",
				Urls: []string{
					versionUrl,
				},
				ImageUpdatedDate: time.Now(),
			},
		},
	}
	demoProject2 = &oneko.Project{
		Uuid:      "63638583-b9d0-4245-8610-e19c040e6e23",
		Name:      "Demo Project 2",
		ImageName: "docker.mycompany.com/demoproject2",
		Versions: []oneko.ProjectVersion{
			{
				Uuid: "5eb9c99f-e1d8-4a70-b394-725de9b4ab12",
				Name: "demoversion2-for-unittest",
				Urls: []string{
					"my-test-instance2.oneko.company.cloud",
				},
				ImageUpdatedDate: time.Now(),
			},
		},
	}
)

func TestMain(m *testing.M) {
	setTestConfiguration()
	uut = New(config.Configuration())
	os.Exit(m.Run())
}

func beforeEach() {
	httpmock.ActivateNonDefault(uut.client.GetClient())
}

func afterEach(t *testing.T) {
	httpmock.DeactivateAndReset()
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
				ApiCallCacheDuration: 15 * time.Second,
			},
			CatnipUrl: "https://catnip.com",
			Mode:      "production",
			Server: config.ServerConfig{
				Port: 8090,
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

	err := uut.ping(context.Background())
	assert.NoError(t, err)
}

func Test_PingReturnsError(t *testing.T) {
	beforeEach()
	defer afterEach(t)

	pingResponse := httpmock.NewStringResponder(401, "")
	httpmock.RegisterResponder("GET", "https://oneko.com/api/session", pingResponse)

	err := uut.ping(context.Background())
	assert.Error(t, err)
}

func Test_Deploy(t *testing.T) {
	beforeEach()
	defer afterEach(t)
	setupProjectResponders(t)

	err := uut.Deploy(projectUuid, versionUuid, context.Background())

	assert.NoError(t, err)

	wakeupCallCount := httpmock.GetCallCountInfo()[fmt.Sprintf("POST /api/project/%s/version/%s/deploy", projectUuid, versionUuid)]

	assert.Equal(t, 1, wakeupCallCount)
}

func Test_GetProjectAndVersionByIds(t *testing.T) {
	beforeEach()
	defer afterEach(t)
	setupProjectResponders(t)

	project, err := uut.GetProjectById(projectUuid, context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, project)

	assert.Equal(t, "Demo Project", project.Name)

	getByIdCallCount := httpmock.GetCallCountInfo()["GET https://oneko.com/api/project/"+projectUuid]

	assert.Equal(t, 1, getByIdCallCount)
}

func Test_HandleRequest_NotFound(t *testing.T) {
	beforeEach()
	defer afterEach(t)

	host := "my-test-instance.oneko.company.cloud"

	httpmock.RegisterResponder("GET", "https://oneko.com/api/project/byDeploymentUrl?deploymentUrl="+host, httpmock.NewStringResponder(404, ""))

	project, err := uut.GetProjectForUrl(host, context.Background())

	assert.EqualError(t, err, "no version matching this url found")
	assert.Nil(t, project)
}

func TestApi_GetAllProjects(t *testing.T) {
	beforeEach()
	defer afterEach(t)
	setupProjectResponders(t)

	projects, err := uut.GetAllProjects(context.Background())

	assert.NoError(t, err)
	assert.Len(t, projects, 2)
}

func setupProjectResponders(t *testing.T) {
	getProjectResponder, err := httpmock.NewJsonResponder(http.StatusOK, demoProject)
	assert.NoError(t, err)
	httpmock.RegisterResponder("GET", fmt.Sprintf("https://oneko.com/api/project/byDeploymentUrl?deploymentUrl=%s", versionUrl), getProjectResponder)
	httpmock.RegisterResponder("GET", fmt.Sprintf("https://oneko.com/api/project/%s", projectUuid), getProjectResponder)

	wakeupResponder := httpmock.NewStringResponder(http.StatusOK, "")
	httpmock.RegisterResponder("POST", fmt.Sprintf("/api/project/%s/version/%s/deploy", projectUuid, versionUuid), wakeupResponder)

	getAllProjectsResponder, err := httpmock.NewJsonResponder(http.StatusOK, []*oneko.Project{demoProject, demoProject2})
	assert.NoError(t, err)
	httpmock.RegisterResponder("GET", "https://oneko.com/api/project", getAllProjectsResponder)
}
