package smpp

import (
    "database/sql"
    "fmt"
    "strings"
    "sync"
    "time"

    _ "github.com/lib/pq"

    "smpp_ussd_gateway/internal/configloader"
    "smpp_ussd_gateway/internal/routing"

    "github.com/fiorix/go-smpp/smpp"
    "github.com/fiorix/go-smpp/smpp/pdu"
    "github.com/fiorix/go-smpp/smpp/pdu/pdufield"
    "github.com/fiorix/go-smpp/smpp/pdu/pdutext"
    "go.uber.org/zap"
)

var logger, _ = zap.NewProduction()
var db *sql.DB

func InitDB(connStr string) error {
    var err error
    db, err = sql.Open("postgres", connStr)
    if err != nil {
        return err
    }
    return db.Ping()
}

func GetDB() *sql.DB {
    return db
}

type SMPPConnection struct {
    Client *smpp.Transceiver
    Addr   string
    Alive  bool
}

type TelcoPool struct {
    Name        string
    Connections map[string]*SMPPConnection
    Mutex       sync.RWMutex
}

var telcoPools = make(map[string]*TelcoPool)

func InitTelcoPools(configs []configloader.TelcoSMPPConfig) {
    for _, cfg := range configs {
        pool := &TelcoPool{
            Name:        cfg.Name,
            Connections: make(map[string]*SMPPConnection),
        }
        for _, ip := range cfg.IPs {
            addr := ipPort(ip, cfg.Port)
            conn, err := createConnection(ip, cfg.Port, cfg.SystemID, cfg.Password, cfg.Name)
            if err != nil {
                logger.Error("Failed to connect", zap.String("addr", addr), zap.Error(err))
                continue
            }
            pool.Connections[addr] = conn
        }
        telcoPools[cfg.Name] = pool
    }
}

func ReloadTelcoConfig(newConfigs []configloader.TelcoSMPPConfig) {
    logger.Info("Reloading telco SMPP configurations...")
    CloseAllPools()         // Close current connections
    InitTelcoPools(newConfigs) // Initialize new connections
}


func createConnection(ip string, port int, sysID, password, telcoName string) (*SMPPConnection, error) {
    addr := ipPort(ip, port)

    // Declare tx first so it can be used inside the handler
    var tx *smpp.Transceiver
    tx = &smpp.Transceiver{
        Addr:         addr,
        User:         sysID,
        Passwd:       password,
        EnquireLink:  10 * time.Second,
        BindInterval: 5 * time.Second,
        Handler: smpp.HandlerFunc(func(p pdu.Body) {
            handleUSSDPDU(tx, p, telcoName)
        }),
    }

    // Bind SMPP connection
    statusChan := tx.Bind()

    // Monitor SMPP connection status
    go func() {
        for status := range statusChan {
            if status.Error() != nil {
                logger.Error("SMPP connection error", zap.String("addr", addr), zap.Error(status.Error()))
            } else {
                logger.Info("SMPP connection status", zap.String("addr", addr), zap.String("status", status.Status().String()))
            }
        }
    }()

    return &SMPPConnection{Client: tx, Addr: addr, Alive: true}, nil
}



func handleUSSDPDU(tx *smpp.Transceiver, pduPacket pdu.Body, telcoName string) {
    if pduPacket.Header().ID != pdu.DeliverSMID {
        return
    }

    startTime := time.Now()
    from := pduPacket.Fields()[pdufield.SourceAddr].String()
    to := pduPacket.Fields()[pdufield.DestinationAddr].String()
    shortMessage := pduPacket.Fields()[pdufield.ShortMessage].String()

    serviceCode := extractServiceCode(shortMessage)
    sessionID := fmt.Sprintf("%d", pduPacket.Header().Seq)

    sessionData := map[string]string{
        "from":       from,
        "to":         to,
        "message":    shortMessage,
        "session_id": sessionID,
        "telco":      telcoName,
    }

    responseText, err := routing.PushUSSDRequest(serviceCode, sessionData)
    if err != nil {
        logger.Error("Failed to push USSD request", zap.String("serviceCode", serviceCode), zap.Error(err))
        return
    }

    _, err = tx.Submit(&smpp.ShortMessage{
        Src:  to,
        Dst:  from,
        Text: pdutext.Raw(responseText),
    })

    endTime := time.Now()
    durationMs := endTime.Sub(startTime).Milliseconds()

    if err != nil {
        logger.Error("Failed to send USSD response", zap.Error(err))
    } else {
        logger.Info("Sent USSD response", zap.String("to", from), zap.String("session_id", sessionID))
    }

    go logTransaction(from, to, shortMessage, responseText, telcoName, sessionID, startTime, endTime, durationMs)
}

func logTransaction(msisdn, shortcode, message, response, telco, sessionID string, start, end time.Time, duration int64) {
    query := `INSERT INTO ussd_logs (msisdn, shortcode, message, response, telco, session_id, start_time, end_time, duration_ms)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

    _, err := db.Exec(query, msisdn, shortcode, message, response, telco, sessionID, start, end, duration)
    if err != nil {
        logger.Error("Failed to log transaction", zap.Error(err))
    } else {
        logger.Info("Transaction logged", zap.String("msisdn", msisdn), zap.Int64("duration_ms", duration))
    }
}

func extractServiceCode(msg string) string {
    if strings.HasPrefix(msg, "*") && strings.HasSuffix(msg, "#") {
        return msg
    }
    parts := strings.Fields(msg)
    if len(parts) > 0 {
        return parts[0]
    }
    return ""
}

func ipPort(ip string, port int) string {
    return fmt.Sprintf("%s:%d", ip, port)
}

func CloseAllPools() {
    for name, pool := range telcoPools {
        pool.Mutex.Lock()
        for addr, conn := range pool.Connections {
            conn.Client.Close()
            logger.Info("Closed connection", zap.String("addr", addr), zap.String("telco", name))
        }
        pool.Mutex.Unlock()
    }
}

