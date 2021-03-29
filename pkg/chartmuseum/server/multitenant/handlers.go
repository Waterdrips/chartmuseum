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
	"github.com/gin-gonic/gin"
)

var (
	objectSavedResponse   = gin.H{"saved": true}
	objectDeletedResponse = gin.H{"deleted": true}
	healthCheckResponse   = gin.H{"healthy": true}
	welcomePageHTML       = []byte(`<!DOCTYPE html>
<html>
<head>
<title>Welcome to ChartMuseum!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to ChartMuseum!</h1>
<p>If you see this page, the ChartMuseum web server is successfully installed and
working.</p>

<p>For online documentation and support please refer to the
<a href="https://github.com/helm/chartmuseum">GitHub project</a>.<br/>

<p><em>Thank you for using ChartMuseum.</em></p>
</body>
</html>
	`)
)

type (
	HTTPError struct {
		Status  int
		Message string
	}
)

type (
	chartOrProvenanceFile struct {
		filename string
		content  []byte
		field    string // file was extracted from this form field
	}
	filenameFromContentFn func([]byte) (string, error)
)

func (server *MultiTenantServer) getWelcomePageHandler(c *gin.Context) {
	c.Data(200, "text/html", welcomePageHTML)
}

func (server *MultiTenantServer) getInfoHandler(c *gin.Context) {
	versionResponse := gin.H{"version": server.Version}
	c.JSON(200, versionResponse)
}

func (server *MultiTenantServer) getHealthCheckHandler(c *gin.Context) {
	c.JSON(200, healthCheckResponse)
}

func (server *MultiTenantServer) getIndexFileRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	log := server.Logger.ContextLoggingFn(c)
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.Data(200, indexFileContentType, indexFile.Raw)
}

func (server *MultiTenantServer) getStorageObjectRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	filename := c.Param("filename")
	log := server.Logger.ContextLoggingFn(c)
	storageObject, err := server.getStorageObject(log, repo, filename)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.Data(200, storageObject.ContentType, storageObject.Content)
}
