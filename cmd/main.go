
package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "smpp_ussd_gateway/internal/api"
    "smpp_ussd_gateway/internal/configloader"
    "smpp_ussd_gateway/internal/smpp"
)

func main() {
    connStr := "postgres://user:password@localhost:5432/ussd_gateway?sslmode=disable"
    err := smpp.InitDB(connStr)
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
