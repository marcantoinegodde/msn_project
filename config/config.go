package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type MSNServerConfiguration struct {
	Database           Database           `mapstructure:"database"`
	DispatchServer     DispatchServer     `mapstructure:"dispatch_server"`
	NotificationServer NotificationServer `mapstructure:"notification_server"`
}

type Database struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type DispatchServer struct {
	ServerAddr             string `mapstructure:"server_addr"`
	ServerPort             string `mapstructure:"server_port"`
	NotificationServerAddr string `mapstructure:"notification_server_addr"`
	NotificationServerPort string `mapstructure:"notification_server_port"`
}

type NotificationServer struct {
	ServerAddr string `mapstructure:"server_addr"`
	ServerPort string `mapstructure:"server_port"`
}

func LoadConfig() (*MSNServerConfiguration, error) {
	var config MSNServerConfiguration

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/msnserver/")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fatal error reading config file: %w", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("fatal error unmarshal config: %w", err)
	}

	return &config, nil
}
