/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package multitenant

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	pathutil "path"
	"testing"
	"time"

	cm_logger "github.com/Waterdrips/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/Waterdrips/chartmuseum/pkg/chartmuseum/router"
	"github.com/chartmuseum/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

// These are generated from scripts/setup-test-environment.sh
var testTarballPath = "../../../../testdata/charts/mychart/mychart-0.1.0.tgz"

type MultiTenantServerTestSuite struct {
	suite.Suite
	Depth0Server         *MultiTenantServer
	Depth1Server         *MultiTenantServer
	Depth2Server         *MultiTenantServer
	Depth3Server         *MultiTenantServer
	ChartURLServer       *MultiTenantServer
	MaxObjectsServer     *MultiTenantServer
	TempDirectory        string
	TestTarballFilename  string
	TestProvfileFilename string
	StorageDirectory     map[string]map[string][]string
	LastCrashMessage     string
	LastPrinted          string
	LastExitCode         int
}

func (suite *MultiTenantServerTestSuite) doRequest(stype string, method string, urlStr string, body io.Reader, contentType string, output ...*bytes.Buffer) gin.ResponseWriter {
	recorder := httptest.NewRecorder()
	if len(output) > 0 {
		recorder.Body = output[0]
	}
	c, _ := gin.CreateTestContext(recorder)
	c.Request, _ = http.NewRequest(method, urlStr, body)
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}

	switch stype {
	case "depth0":
		suite.Depth0Server.Router.HandleContext(c)
	case "depth1":
		suite.Depth1Server.Router.HandleContext(c)
	case "depth2":
		suite.Depth2Server.Router.HandleContext(c)
	case "depth3":
		suite.Depth3Server.Router.HandleContext(c)
	case "charturl":
		suite.ChartURLServer.Router.HandleContext(c)
	case "maxobjects":
		suite.MaxObjectsServer.Router.HandleContext(c)
	}

	return c.Writer
}

func (suite *MultiTenantServerTestSuite) copyTestFilesTo(dir string) {
	srcFileTarball, err := os.Open(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	defer srcFileTarball.Close()

	destFileTarball, err := os.Create(pathutil.Join(dir, "mychart-0.1.0.tgz"))
	suite.Nil(err, fmt.Sprintf("no error creating new tarball in %s", dir))
	defer destFileTarball.Close()

	_, err = io.Copy(destFileTarball, srcFileTarball)
	suite.Nil(err, fmt.Sprintf("no error copying test testball to %s", dir))

	err = destFileTarball.Sync()
	suite.Nil(err, fmt.Sprintf("no error syncing temp tarball in %s", dir))
}

func (suite *MultiTenantServerTestSuite) populateOrgTeamRepoDirectory(org string, team string, repo string) {
	testPrefix := fmt.Sprintf("%s/%s/%s", org, team, repo)
	newDir := pathutil.Join(suite.TempDirectory, testPrefix)
	os.MkdirAll(newDir, os.ModePerm)
	suite.copyTestFilesTo(newDir)
	suite.copyTestFilesTo(pathutil.Join(newDir, ".."))
	suite.copyTestFilesTo(pathutil.Join(newDir, "../.."))
}

func (suite *MultiTenantServerTestSuite) SetupSuite() {
	timestamp := time.Now().Format("20060102150405")
	suite.TempDirectory = fmt.Sprintf("../../../../.test/chartmuseum-multitenant-server/%s", timestamp)
	os.MkdirAll(suite.TempDirectory, os.ModePerm)
	suite.copyTestFilesTo(suite.TempDirectory)

	srcFileTarball, err := os.Open(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	defer srcFileTarball.Close()

	suite.TestTarballFilename = pathutil.Join(suite.TempDirectory, "mychart-0.1.0.tgz")
	destFileTarball, err := os.Create(suite.TestTarballFilename)
	suite.Nil(err, "no error creating new tarball in temp dir")
	defer destFileTarball.Close()

	_, err = io.Copy(destFileTarball, srcFileTarball)
	suite.Nil(err, "no error copying test testball to temp tarball")

	err = destFileTarball.Sync()
	suite.Nil(err, "no error syncing temp tarball")

	suite.StorageDirectory = map[string]map[string][]string{
		"org1": {
			"team1": {"repo1", "repo2", "repo3"},
			"team2": {"repo1", "repo2", "repo3"},
			"team3": {"repo1", "repo2", "repo3"},
		},
		"org2": {
			"team1": {"repo1", "repo2", "repo3"},
			"team2": {"repo1", "repo2", "repo3"},
			"team3": {"repo1", "repo2", "repo3"},
		},
		"org3": {
			"team1": {"repo1", "repo2", "repo3"},
			"team2": {"repo1", "repo2", "repo3"},
			"team3": {"repo1", "repo2", "repo3"},
		},
	}

	// Scaffold out test storage directory structure
	for org, teams := range suite.StorageDirectory {
		for team, repos := range teams {
			for _, repo := range repos {
				suite.populateOrgTeamRepoDirectory(org, team, repo)
			}
		}
	}

	backend := storage.Backend(storage.NewLocalFilesystemBackend(suite.TempDirectory))

	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug: true,
	})
	suite.Nil(err, "no error creating logger")

	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		EnableMetrics: true,
	})
	server, err := NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		IndexLimit:             1,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=0) server")
	suite.Depth0Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         1,
		EnableMetrics: true,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=1) server")
	suite.Depth1Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         2,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=2) server")
	suite.Depth2Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         3,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=3) server")
	suite.Depth3Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		ChartURL:               "https://chartmuseum.com",
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new custom chart URL server")
	suite.ChartURLServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		MaxStorageObjects:      1,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new max objects server")
	suite.MaxObjectsServer = server
}

func (suite *MultiTenantServerTestSuite) TearDownSuite() {
	os.RemoveAll(suite.TempDirectory)
}

func (suite *MultiTenantServerTestSuite) regenerateRepositoryIndex(repo string, isFound bool) {
	server := suite.Depth0Server
	if repo != "" {
		server = suite.Depth1Server
	}
	log := server.Logger.ContextLoggingFn(&gin.Context{})

	entry, err := server.initCacheEntry(log, repo)
	suite.Nil(err, "no error on init cache entry")

	objects, err := server.fetchChartsInStorage(log, repo)
	if !isFound {
		suite.Equal(len(objects), 0)
		return
	}
	suite.Nil(err, "no error on fetchChartsInStorage")
	diff := storage.GetObjectSliceDiff(server.getRepoObjectSlice(entry), objects, server.TimestampTolerance)
	_, err = server.regenerateRepositoryIndexWorker(log, entry, diff)
	suite.Nil(err, "no error regenerating repo index")

	newtime := time.Now().Add(1 * time.Hour)
	err = os.Chtimes(suite.TestTarballFilename, newtime, newtime)
	suite.Nil(err, "no error changing modtime on temp file")

	objects, err = server.fetchChartsInStorage(log, repo)
	suite.Nil(err, "no error on fetchChartsInStorage")
	diff = storage.GetObjectSliceDiff(server.getRepoObjectSlice(entry), objects, server.TimestampTolerance)
	_, err = server.regenerateRepositoryIndexWorker(log, entry, diff)
	suite.Nil(err, "no error regenerating repo index with tarball updated")
}
func (suite *MultiTenantServerTestSuite) TestRegenerateRepositoryIndex() {
	suite.regenerateRepositoryIndex("", true)
	suite.regenerateRepositoryIndex("org1", true)
	suite.regenerateRepositoryIndex("not-set-org", false)
}

func (suite *MultiTenantServerTestSuite) TestGenIndex() {
	_, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug:   true,
		LogJSON: true,
	})
	suite.Nil(err, "no error creating logger")
}

func (suite *MultiTenantServerTestSuite) TestCustomChartURLServer() {
	res := suite.doRequest("charturl", "GET", "/index.yaml", nil, "")
	suite.Equal(200, res.Status(), "200 GET /index.yaml")
}

func (suite *MultiTenantServerTestSuite) TestRoutes() {
	suite.testAllRoutes("", 0)
	for org, teams := range suite.StorageDirectory {
		suite.testAllRoutes(org, 1)
		for team, repos := range teams {
			suite.testAllRoutes(pathutil.Join(org, team), 2)
			for _, repo := range repos {
				suite.testAllRoutes(pathutil.Join(org, team, repo), 3)
			}
		}
	}
}

func (suite *MultiTenantServerTestSuite) testAllRoutes(repo string, depth int) {
	var res gin.ResponseWriter

	stype := fmt.Sprintf("depth%d", depth)

	// GET /
	res = suite.doRequest(stype, "GET", "/", nil, "")
	suite.Equal(200, res.Status(), "200 GET /")

	// GET /health
	res = suite.doRequest(stype, "GET", "/health", nil, "")
	suite.Equal(200, res.Status(), "200 GET /health")

	var repoPrefix string
	if repo != "" {
		repoPrefix = pathutil.Join("/", repo)
	} else {
		repoPrefix = ""
	}

	// GET /:repo/index.yaml
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/index.yaml", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/index.yaml", repoPrefix))

	// Issue #21
	suite.NotEqual("", res.Header().Get("X-Request-Id"), "X-Request-Id header is present")
	suite.Equal("", res.Header().Get("X-Blah-Blah-Blah"), "X-Blah-Blah-Blah header is not present")

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart-0.1.0", repoPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart-0.1.0", repoPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/index.yaml", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/index.yaml", repoPrefix))
}

func TestMultiTenantServerTestSuite(t *testing.T) {
	suite.Run(t, new(MultiTenantServerTestSuite))
}
