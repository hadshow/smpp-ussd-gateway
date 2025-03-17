package configloader

import (
    "fmt"
    "os"
    "time"

    "github.com/fsnotify/fsnotify"
    "github.com/spf13/viper"
)

type TelcoSMPPConfig struct {
    Name      string   `mapstructure:"name"`
    IPs       []string `mapstructure:"ips"`
    Port      int      `mapstructure:"port"`
    SystemID  string   `mapstructure:"system_id"`
    Password  string   `mapstructure:"password"`
    BindType  string   `mapstructure:"bind_type"`
}

type RouteRule struct {
    ServiceCode string            `mapstructure:"service_code"`
    Method      string            `mapstructure:"method"`
    URL         string            `mapstructure:"url"`
    Headers     map[string]string `mapstructure:"headers"`
}

type Config struct {
    Telcos []TelcoSMPPConfig `mapstructure:"telcos"`
    Routes []RouteRule       `mapstructure:"routes"`
}

func LoadConfig(filePath string) (Config, error) {
    viper.SetConfigFile(filePath)
    var config Config

    if err := viper.ReadInConfig(); err != nil {
        return config, err
    }

