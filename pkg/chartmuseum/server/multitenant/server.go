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
	"sync"
	"time"

	cm_logger "github.com/Waterdrips/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/Waterdrips/chartmuseum/pkg/chartmuseum/router"
	cm_repo "github.com/Waterdrips/chartmuseum/pkg/repo"
	"github.com/Waterdrips/chartmuseum/pkg/cache"

	"github.com/chartmuseum/storage"
	cm_storage "github.com/chartmuseum/storage"
)

type (
	// MultiTenantServer contains a Logger, Router, storage backend and object cache
	MultiTenantServer struct {
		Logger                 *cm_logger.Logger
		Router                 *cm_router.Router
		StorageBackend         storage.Backend
		TimestampTolerance     time.Duration
		ExternalCacheStore     cache.Store
		InternalCacheStore     map[string]*cacheEntry
		MaxStorageObjects      int
		IndexLimit             int
		UseStatefiles          bool
		EnforceSemver2         bool
		ChartURL               string
		Version                string
		Limiter                chan struct{}
		Tenants                map[string]*tenantInternals
		TenantCacheKeyLock     *sync.Mutex
		CacheInterval          time.Duration
		EventChan              chan event
	}

	// MultiTenantServerOptions are options for constructing a MultiTenantServer
	MultiTenantServerOptions struct {
		Logger                 *cm_logger.Logger
		Router                 *cm_router.Router
		StorageBackend         storage.Backend
		ExternalCacheStore     cache.Store
		TimestampTolerance     time.Duration
		ChartURL               string
		Version                string
		MaxStorageObjects      int
		IndexLimit             int
		GenIndex               bool
		UseStatefiles          bool
		CacheInterval          time.Duration
	}

	tenantInternals struct {
		FetchedObjectsLock      *sync.Mutex
		RegenerationLock        *sync.Mutex
		FetchedObjectsChans     []chan fetchedObjects
		RegeneratedIndexesChans []chan indexRegeneration
	}

	fetchedObjects struct {
		objects []cm_storage.Object
		err     error
	}

	indexRegeneration struct {
		index *cm_repo.Index
		err   error
	}
)

// NewMultiTenantServer creates a new MultiTenantServer instance
func NewMultiTenantServer(options MultiTenantServerOptions) (*MultiTenantServer, error) {
	var chartURL string
	if options.ChartURL != "" {
		chartURL = options.ChartURL + options.Router.ContextPath
	}

	server := &MultiTenantServer{
		Logger:                 options.Logger,
		Router:                 options.Router,
		StorageBackend:         options.StorageBackend,
		TimestampTolerance:     options.TimestampTolerance,
		ExternalCacheStore:     options.ExternalCacheStore,
		InternalCacheStore:     map[string]*cacheEntry{},
		MaxStorageObjects:      options.MaxStorageObjects,
		IndexLimit:             options.IndexLimit,
		ChartURL:               chartURL,
		UseStatefiles:          options.UseStatefiles,
		Version:                options.Version,
		Limiter:                make(chan struct{}, options.IndexLimit),
		Tenants:                map[string]*tenantInternals{},
		TenantCacheKeyLock:     &sync.Mutex{},
		CacheInterval:          options.CacheInterval,
	}

	server.Router.SetRoutes(server.Routes())

	server.EventChan = make(chan event, server.IndexLimit)
	go server.startEventListener()
	server.initCacheTimer()

	return server, nil
}

// Listen starts the router on a given port
func (server *MultiTenantServer) Listen(port int) {
	server.Router.Start(port)
}
