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

package main

import (
	"errors"
	"fmt"
	"github.com/Waterdrips/chartmuseum/pkg/chartmuseum"
	"os"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
	RedisMock        *miniredis.Miniredis
	LastCrashMessage string
}

func (suite *MainTestSuite) SetupSuite() {
	crash = func(v ...interface{}) {
		suite.LastCrashMessage = fmt.Sprint(v...)
		panic(v)
	}
	newServer = func(options chartmuseum.ServerOptions) (chartmuseum.Server, error) {
		return nil, errors.New("graceful crash")
	}

	redisMock, err := miniredis.Run()
	suite.Nil(err, "able to create miniredis instance")
	suite.RedisMock = redisMock
}

func (suite *MainTestSuite) TearDownSuite() {
	suite.RedisMock.Close()
}

func (suite *MainTestSuite) TestMain() {
	os.Args = []string{"chartmuseum", "--config", "blahblahblah.yaml"}
	suite.Panics(main, "bad config")
	suite.Equal("config file \"blahblahblah.yaml\" does not exist", suite.LastCrashMessage, "crashes with bad config")

	os.Args = []string{"chartmuseum"}
	suite.Panics(main, "no storage")
	suite.Equal("Missing required flags(s): --storage", suite.LastCrashMessage, "crashes with no storage")

}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
