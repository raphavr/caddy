package auth

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/raphavr/caddy"
	"github.com/raphavr/caddy/caddyhttp/httpserver"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

const (
	pluginName = "auth"
	serverType = "http"
	tokenKey   = "token"

	viperProviderKey   = "VIPER_PROVIDER"
	viperEndpointKey   = "VIPER_ENDPOINT"
	viperPathKey       = "VIPER_PATH"
	viperConfigTypeKey = "VIPER_CONFIG_TYPE_KEY"

	viperProviderDefault   = "consul"
	viperEndpointDefault   = "localhost:8500"
	viperPathDefault       = "config/gold-proxy"
	viperConfigTypeDefault = "json"
)

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
	viper.AddRemoteProvider(
		getEnv(viperProviderKey, viperProviderDefault),
		getEnv(viperEndpointKey, viperEndpointDefault),
		getEnv(viperPathKey, viperPathDefault),
	)

	viper.SetConfigType(getEnv(viperConfigTypeKey, viperConfigTypeDefault))

	err := load()
	if err != nil {
		log.Print("[WARNING] Impossible to get auth config")
	}
	go reload(configReloadInterval)
}

func load() error {
	err := viper.ReadRemoteConfig()
	if err != nil {
		return err
	}

	var m map[string]interface{}

	err = viper.Unmarshal(&m)
	if err != nil {
		return err
	}

	if token, found := m[tokenKey].(string); found {
		store.Lock()
		store.token = token
		store.Unlock()
	} else {
		return errors.New("Token key not found")
	}

	return nil
}

func reload(interval time.Duration) {
	for {
		time.Sleep(interval)

		err := viper.ReadRemoteConfig()
		if err != nil {
			log.Print("[WARNING] Impossible to get auth config")
			continue
		}

		var m map[string]interface{}

		err = viper.Unmarshal(&m)
		if err != nil {
			log.Print("[WARNING] Impossible to unmarshal auth config")
			continue
		}

		if token, found := m[tokenKey].(string); found {
			store.Lock()
			store.token = token
			store.Unlock()
		} else {
			log.Print("[WARNING] Token key not found")
		}
	}
}
