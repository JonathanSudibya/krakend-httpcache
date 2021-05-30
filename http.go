// Package httpcache introduces an in-memory-cached http client into the KrakenD stack
package httpcache

import (
	"context"
	"net/http"

	"github.com/gomodule/redigo/redis"
	"github.com/gregjones/httpcache"
	cacheRedis "github.com/gregjones/httpcache/redis"
	"github.com/luraproject/lura/config"
	"github.com/luraproject/lura/proxy"
	"github.com/luraproject/lura/transport/http/client"
)

// Namespace is the key to use to store and access the custom config data
const Namespace = "github.com/jonathansudibya/krakend-httpcache"

var (
	memTransport = httpcache.NewMemoryCacheTransport()
	httpClient   http.Client
)

// NewHTTPClient creates a HTTPClientFactory using an in-memory-cached http client
func NewHTTPClient(cfg *config.Backend) client.HTTPClientFactory {
	extCfgRaw, ok := cfg.ExtraConfig[Namespace]
	if !ok {
		return client.NewHTTPClient
	}
	return func(_ context.Context) *http.Client {
		extCfg, ok := extCfgRaw.(map[string]interface{})
		if !ok {
			return &http.Client{Transport: memTransport}
		}

		// get storage type
		// enum : "memory", "redis"
		storage, ok := extCfg["storage"].(string)
		if !ok {
			storage = "memory"
		}

		switch storage {
		case "redis":
			hostname, ok := extCfg["redis_hostname"].(string)
			if !ok {
				hostname = ":6379"
			}

			db, ok := extCfg["redis_db"].(int)
			if !ok {
				db = 0
			}

			redisPassword, ok := extCfg["redis_password"].(string)
			if !ok {
				redisPassword = ""
			}

			redisUsername, ok := extCfg["redis_username"].(string)
			if !ok {
				redisUsername = ""
			}

			dbConn, err := redis.Dial("tcp", hostname, redis.DialDatabase(db), redis.DialPassword(redisPassword), redis.DialUsername(redisUsername))
			if err != nil {
				break
			}

			cacheStrg := cacheRedis.NewWithClient(dbConn)
			redisTransport := httpcache.NewTransport(cacheStrg)
			httpClient = http.Client{Transport: redisTransport}
		default:
			httpClient = http.Client{Transport: memTransport}
		}

		return &httpClient
	}
}

// BackendFactory returns a proxy.BackendFactory that creates backend proxies using
// an in-memory-cached http client
func BackendFactory(cfg *config.Backend) proxy.BackendFactory {
	return proxy.CustomHTTPProxyFactory(NewHTTPClient(cfg))
}
