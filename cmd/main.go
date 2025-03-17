package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/joho/godotenv" // NEW for .env support
    "smpp_ussd_gateway/internal/api"
    "smpp_ussd_gateway/internal/configloader"
    "smpp_ussd_gateway/internal/smpp"
)

func main() {
    // Load .env file
    err := godotenv.Load()
    if err != nil {
        fmt.Println("Error loading .env file")
        os.Exit(1)
    }

    // Load DB connection string from .env
    connStr := os.Getenv("DB_CONN_STRING")
    if connStr == "" {
        fmt.Println("Environment variable DB_CONN_STRING is not set")
        os.Exit(1)
    }

    err = smpp.InitDB(connStr)
    if err != nil {
        panic("DB connection failed: " + err.Error())
    }

    api.InitAPI(smpp.GetDB())
    go api.StartServer()

    config, err := configloader.LoadConfig("config/config.yaml")
    if err != nil {
        fmt.Printf("Failed to load config: %v\n", err)
        os.Exit(1)
    }

    smpp.InitTelcoPools(config.Telcos)
    configloader.WatchConfigFile("config/config.yaml", func(newConfig configloader.Config) {
        smpp.ReloadTelcoConfig(newConfig.Telcos)
    })

    fmt.Println("SMPP USSD Gateway is running...")
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    <-sigs
    fmt.Println("Shutting down SMPP Gateway...")
    smpp.CloseAllPools()
    fmt.Println("Shutdown complete.")
}

