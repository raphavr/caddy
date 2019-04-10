package auth

import (
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

	viperProviderKey   = "VIPER_PROVIDER"
	viperEndpointKey   = "VIPER_ENDPOINT"
	viperPathKey       = "VIPER_PATH"
	viperConfigTypeKey = "VIPER_CONFIG_TYPE_KEY"

	viperProviderDefault   = "consul"
	viperEndpointDefault   = "localhost:8500"
	viperPathDefault       = "config/gold-proxy/token"
	viperConfigTypeDefault = "VIPER_CONFIG_TYPE_KEY"
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

	viper.SetConfigType("json")

	err := load()
	if err != nil {
		panic(err)
	}
	go reload(configReloadInterval)
}

func load() error {
	err := viper.ReadRemoteConfig()
	if err != nil {
		return err
	}

	store.RLock()
	currentToken := store.token
	store.RUnlock()

	var m map[string]interface{}

	err = viper.Unmarshal(&m)
	if err != nil {
		return err
	}

	store.Lock()
	store.token = currentToken
	store.Unlock()
	return nil
}

func reload(interval time.Duration) {
	for {
		time.Sleep(interval)

		err := viper.ReadRemoteConfig()
		if err != nil {
			continue
		}

		store.RLock()
		currentToken := store.token
		store.RUnlock()

		err = viper.Unmarshal(&currentToken)
		if err != nil {
			continue
		}

		store.Lock()
		store.token = currentToken
		store.Unlock()
	}
}
