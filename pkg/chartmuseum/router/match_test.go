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

package router

import (
	"net/http/httptest"
	pathutil "path"
	"testing"

	cm_auth "github.com/chartmuseum/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type MatchTestSuite struct {
	suite.Suite
}

func (suite *MatchTestSuite) TestMatch() {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handlers := []gin.HandlerFunc{}

	for i := 0; i <= 9; i++ {
		{
			j := i
			handlers = append(handlers, func(c *gin.Context) {
				c.Set("index", j)
			})
		}
	}

	routes := []*Route{
		{"GET", "/", handlers[0], cm_auth.PullAction},
		{"GET", "/health", handlers[1], ""},
		{"GET", "/:repo/index.yaml", handlers[2], cm_auth.PullAction},
		{"GET", "/:repo/charts/:filename", handlers[3], cm_auth.PullAction},
	}

	for depth := 0; depth <= 3; depth++ {
		var repo string

		switch {
		case depth == 1:
			repo = "myrepo"
		case depth == 2:
			repo = "myorg/myrepo"
		case depth == 3:
			repo = "myorg/myteam/myrepo"
		}

		for _, contextPath := range []string{"", "/x", "/x/y", "/x/y/z"} {

			// GET /
			r := pathutil.Join("/", contextPath)
			route, params := match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic := match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			suite.Nil(params)
			if route != nil {
				route.Handler(c)
			}
			val, exists := c.Get("index")
			suite.True(exists)
			suite.Equal(0, val)

			// GET /health
			r = pathutil.Join("/", contextPath, "health")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			suite.Nil(params)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(1, val)

			// GET /index.yaml
			r = pathutil.Join("/", contextPath, repo, "index.yaml")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(2, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// GET /charts/mychart-0.1.0.tgz
			r = pathutil.Join("/", contextPath, repo, "charts/mychart-0.1.0.tgz")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(3, val)
			suite.Equal([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", repo}}, params)

		}
	}

	// Test route repos named "api*"
	r := "/apix/index.yaml"
	route, params := match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic := match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists := c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal([]gin.Param{{"repo", "apix"}}, params)

	r = "/apix/charts/mychart-0.1.0.tgz"
	route, params = match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(3, val)
	suite.Equal([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", "apix"}}, params)

	// Test route repos named "health"
	r = "/health/index.yaml"
	route, params = match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal([]gin.Param{{"repo", "health"}}, params)

	r = "/health/charts/mychart-0.1.0.tgz"
	route, params = match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(3, val)
	suite.Equal([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", "health"}}, params)
}

func TestMatchTestSuite(t *testing.T) {
	suite.Run(t, new(MatchTestSuite))
}
