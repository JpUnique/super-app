package config

import (
	"log"

	"github.com/spf13/viper"
)

type GatewayConfig struct {
	Port string `mapstructure:"port"`

	RedisURL string `mapstructure:"redis_url"`

	JWTPrivateKey string `mapstructure:"jwt_private_key"`
	JWTPublicKey  string `mapstructure:"jwt_public_key"`

	APIKeyHeader string `mapstructure:"api_key_header"`
	JWTHeader    string `mapstructure:"jwt_header"`
	RoutesFile  string `mapstructure:"routes_file"`
}

func Load() GatewayConfig {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./internal/gateway/config")
    viper.AddConfigPath("../internal/gateway/config")
    viper.AddConfigPath(".")


	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("config error: %v", err)
	}

	var cfg GatewayConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("config parse error: %v", err)
	}

    if cfg.RoutesFile == "" {
        cfg.RoutesFile = "./internal/gateway/proxy/routes.yaml"
    }

	return cfg
}
