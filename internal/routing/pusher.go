package routing

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "strings"

    "go-smpp-ussd-gateway/internal/configloader"
    "go.uber.org/zap"
)

var logger, _ = zap.NewProduction()
var RouteMap = make(map[string]configloader.RouteRule)

func InitRoutes(routes []configloader.RouteRule) {
    temp := make(map[string]configloader.RouteRule)
    for _, route := range routes {
        key := strings.TrimSpace(route.ServiceCode)
        temp[key] = route
    }
    RouteMap = temp
    logger.Info("Route rules initialized", zap.Int("count", len(RouteMap)))
}

func PushUSSDRequest(serviceCode string, sessionData map[string]string) (string, error) {
    route, exists := RouteMap[serviceCode]
    if !exists {
        return "", fmt.Errorf("no route found for service code: %s", serviceCode)
    }

    client := &http.Client{}
    var req *http.Request
    var err error

    if route.Method == "GET" {
        params := "?"
        for k, v := range sessionData {
            params += fmt.Sprintf("%s=%s&", k, v)
        }
        url := route.URL + strings.TrimRight(params, "&")
        req, err = http.NewRequest("GET", url, nil)
    } else {
        body, _ := json.Marshal(sessionData)
        req, err = http.NewRequest("POST", route.URL, bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
    }

    if err != nil {
        return "", err
    }

    for k, v := range route.Headers {
        req.Header.Set(k, v)
    }

    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    respBody, _ := ioutil.ReadAll(resp.Body)
    logger.Info("USSD 

