package auth

import (
	"log"
	"os"
	"time"

	consulapi "github.com/armon/consul-api"
	"github.com/raphavr/caddy"
	"github.com/raphavr/caddy/caddyhttp/httpserver"
)

const (
	pluginName = "auth"
	serverType = "http"
	tokenKey   = "token"

	consulEndpointKey   = "CONSUL_ENDPOINT"
	consulPathKey       = "CONSUL_PATH"
	consulDataCenterKey = "CONSUL_DATA_CENTER"

	consulEndpointDefault   = "consul:8500"
	consulPathDefault       = "config/gold-proxy/token"
	consulDataCenterDefault = "dc1"
)

var consulClient *consulapi.Client

func setup(c *caddy.Controller) error {
	urlPattern, err := parseURLPattern(c)
	if err != nil {
		return err
	}

	httpserver.GetConfig(c).AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return authHandler{Next: next, URLPattern: *urlPattern}
	})

	return nil
}

func parseURLPattern(c *caddy.Controller) (*string, error) {
	c.Next()
	if !c.NextArg() {
		return nil, c.ArgErr()
	}
	token := c.Val()
	return &token, nil
}

func init() {
	mustInitConfig(time.Duration(1) * time.Minute)

	caddy.RegisterPlugin(pluginName, caddy.Plugin{
		ServerType: serverType,
		Action:     setup,
	})
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func mustInitConfig(configReloadInterval time.Duration) {
	config := consulapi.DefaultConfig()
	config.Address = getEnv(consulEndpointKey, consulEndpointDefault)
	config.Datacenter = getEnv(consulDataCenterKey, consulDataCenterDefault)

	var err error
	consulClient, err = consulapi.NewClient(config)
	if err != nil {
		panic(err)
	}

	err = load()
	if err != nil {
		log.Print("[WARNING] Impossible to get config from consul:" + err.Error())
	}
	go reload(configReloadInterval)
}

func load() error {
	v, _, err := consulClient.KV().Get(getEnv(consulPathKey, consulPathDefault), nil)
	if err != nil {
		return err
	}

	if v == nil {
		store.Lock()
		store.token = ""
		store.Unlock()
		return nil
	}

	store.Lock()
	store.token = string(v.Value)
	store.Unlock()
	return nil
}

func reload(interval time.Duration) {
	for {
		time.Sleep(interval)
		load()
	}
}
