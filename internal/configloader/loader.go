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
    if err := viper.Unmarshal(&config); err != nil {
        return config, err
    }
    return config, nil
}

func WatchConfigFile(filePath string, onChange func(Config)) {
    v := viper.New()
    v.SetConfigFile(filePath)
    if err := v.ReadInConfig(); err != nil {
        fmt.Println("Failed to read config for watch:", err)
        return
    }

    v.WatchConfig()
    v.OnConfigChange(func(e fsnotify.Event) {
        time.Sleep(500 * time.Millisecond)
        var config Config
        if err := v.Unmarshal(&config); err != nil {
            fmt.Println("Failed to reload config:", err)
            return
        }
        fmt.Println("Config file changed, reloading...")
        onChange(config)
    })

    go func() {
        for {
            if _, err := os.Stat(filePath); err != nil {
                fmt.Println("Config file not found, stopping watcher.")
                return
            }
            time.Sleep(10 * time.Second)
        }
    }()
}

