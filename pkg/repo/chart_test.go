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

package repo

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chart"
	helm_repo "helm.sh/helm/v3/pkg/repo"
)

type ChartTestSuite struct {
	suite.Suite
	TarballContent []byte
}

func (suite *ChartTestSuite) SetupSuite() {
	tarballPath := "../../testdata/charts/mychart/mychart-0.1.0.tgz"
	content, err := ioutil.ReadFile(tarballPath)
	suite.Nil(err, "no error reading test tarball")
	suite.TarballContent = content
}

func (suite *ChartTestSuite) TestChartPackageFilenameFromNameVersion() {
	filename := ChartPackageFilenameFromNameVersion("mychart", "2.3.4")
	suite.Equal("mychart-2.3.4", filename, "filename as expected")
}

func (suite *ChartTestSuite) TestStorageObjectFromChartVersion() {
	now := time.Now()
	chartVersion := &helm_repo.ChartVersion{
		Metadata: &chart.Metadata{
			Name:    "mychart",
			Version: "0.1.0",
		},
		URLs:    []string{"charts/mychart-0.1.0"},
		Created: now,
	}
	object := StorageObjectFromChartVersion(chartVersion)
	suite.Equal(now, object.LastModified, "object last modified as expected")
	suite.Equal("mychart-0.1.0", object.Path, "object path as expected")
	suite.Equal("mychart", object.Meta.Name, "object chart name as expected")
	suite.Equal("0.1.0", object.Meta.Version, "object chart version as expected")
	suite.Equal([]byte{}, object.Content, "object content as expected")
}

func TestChartTestSuite(t *testing.T) {
	suite.Run(t, new(ChartTestSuite))
}
