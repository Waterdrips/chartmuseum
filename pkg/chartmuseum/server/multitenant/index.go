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
	"net/http"

	cm_logger "github.com/Waterdrips/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "github.com/Waterdrips/chartmuseum/pkg/repo"
	cm_storage "github.com/chartmuseum/storage"
)

var (
	indexFileContentType = "application/x-yaml"
)

func (server *MultiTenantServer) getIndexFile(log cm_logger.LoggingFn, repo string) (*cm_repo.Index, *HTTPError) {
	entry, err := server.initCacheEntry(log, repo)
	if err != nil {
		errStr := err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return nil, &HTTPError{http.StatusInternalServerError, errStr}
	}

	// if cache is nil, and not on a timer, regenerate it
	if len(entry.RepoIndex.Entries) == 0 && server.CacheInterval == 0 {

		fo := <-server.getChartList(log, repo)

		if fo.err != nil {
			errStr := fo.err.Error()
			log(cm_logger.ErrorLevel, errStr,
				"repo", repo,
			)
			return nil, &HTTPError{http.StatusInternalServerError, errStr}
		}

		objects := server.getRepoObjectSlice(entry)
		diff := cm_storage.GetObjectSliceDiff(objects, fo.objects, server.TimestampTolerance)

		// return fast if no changes
		if !diff.Change {
			log(cm_logger.DebugLevel, "No change detected between cache and storage",
				"repo", repo,
			)
		} else {
			ir := <-server.regenerateRepositoryIndex(log, entry, diff)
			if ir.err != nil {
				errStr := ir.err.Error()
				log(cm_logger.ErrorLevel, errStr,
					"repo", repo,
				)
				return ir.index, &HTTPError{http.StatusInternalServerError, errStr}
			}
			entry.RepoIndex = ir.index
		}
	}
	return entry.RepoIndex, nil
}

func (server *MultiTenantServer) getRepoObjectSlice(entry *cacheEntry) []cm_storage.Object {
	var objects []cm_storage.Object
	for _, entry := range entry.RepoIndex.Entries {
		for _, chartVersion := range entry {
			object := cm_repo.StorageObjectFromChartVersion(chartVersion)
			objects = append(objects, object)
		}
	}
	return objects
}
