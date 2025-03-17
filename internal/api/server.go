package api

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"

    "go.uber.org/zap"
)

var logger, _ = zap.NewProduction()
var db *sql.DB

func InitAPI(database *sql.DB) {
    db = database
}

func StartServer() {
    http.HandleFunc("/api/reports/summary", summaryHandler)
    http.HandleFunc("/api/logs/by-msisdn", logsByMsisdnHandler)
    http.HandleFunc("/api/logs/by-date", logsByDateHandler)
    logger.Info("REST API server started on :9080")
    http.ListenAndServe(":9080", nil)
}

func summaryHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query(`
        SELECT telco, COUNT(*) as session_count, AVG(duration_ms) as avg_duration
        FROM ussd_logs
        GROUP BY telco
    `)
    if err != nil {
        http.Error(w, "DB query failed", 500)
        logger.Error("DB query failed", zap.Error(err))
        return
    }
    defer rows.Close()

    type Report struct {
        Telco         string  `json:"telco"`
        SessionCount  int     `json:"session_count"`
        AvgDurationMs float64 `json:"avg_duration_ms"`
    }

    var reports []Report
    for rows.Next() {
        var r Report
        rows.Scan(&r.Telco, &r.SessionCount, &r.AvgDurationMs)
        reports = append(reports, r)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(reports)
}

func logsByMsisdnHandler(w http.ResponseWriter, r *http.Request) {
    msisdn := r.URL.Query().Get("msisdn")
    if msisdn == "" {
        http.Error(w, "msisdn is required", 400)
        return
    }

    rows, err := db.Query(`
        SELECT msisdn, shortcode, message, response, telco, session_id, start_time, end_time, duration_ms
        FROM ussd_logs
        WHERE msisdn = $1
        ORDER BY start_time DESC
        LIMIT 100`, msisdn)

    if err != nil {
        http.Error(w, "DB query failed", 500)
        logger.Error("DB query failed", zap.Error(err))
        return
    }
    defer rows.Close()

    type LogEntry struct {
        Msisdn     string    `json:"msisdn"`
        Shortcode  string    `json:"shortcode"`
        Message    string    `json:"message"`
        Response   string    `json:"response"`
        Telco      string    `json:"telco"`
        SessionID  string    `json:"session_id"`
        StartTime  time.Time `json:"start_time"`
        EndTime    time.Time `json:"end_time"`
        DurationMs int       `json:"duration_ms"`
    }

    var logs []LogEntry
    for rows.Next() {
        var entry LogEntry
        rows.Scan(&entry.Msisdn, &entry.Shortcode, &entry.Message, &entry.Response, &entry.Telco, &entry.SessionID, &entry.StartTime, &entry.EndTime, &entry.DurationMs)
        logs = append(logs, entry)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(logs)
}

func logsByDateHandler(w http.ResponseWriter, r *http.Request) {
    startDate := r.URL.Query().Get("start")
    endDate := r.URL.Query().Get("end")

    if startDate == "" || endDate == "" {
        http.Error(w, "start and end dates are required (YYYY-MM-DD)", 400)
        return
    }

    rows, err := db.Query(`
        SELECT msisdn, shortcode, message, response, telco, session_id, start_time, end_time, duration_ms
        FROM ussd_logs
        WHERE start_time BETWEEN $1 AND $2
        ORDER BY start_time DESC
        LIMIT 100`, startDate, endDate)

    if err != nil {
        http.Error(w, "DB query failed", 500)
        logger.Error("DB query failed", zap.Error(err))
        return
    }
    defer rows.Close()

    type LogEntry struct {
        Msisdn     string    `json:"msisdn"`
        Shortcode  string    `json:"shortcode"`
        Message    string    `json:"message"`
        Response   string    `json:"response"`
        Telco      string    `json:"telco"`
        SessionID  string    `json:"session_id"`
        StartTime  time.Time `json:"start_time"`
        EndTime    time.Time `json:"end_time"`
        DurationMs int       `json:"duration_ms"`
    }

    var logs []LogEntry
    for rows.Next() {
        var entry LogEntry
        rows.Scan(&entry.Msisdn, &entry.Shortcode, &entry.Message, &entry.Response, &entry.Telco, &entry.SessionID, &entry.StartTime, &entry.EndTime, &entry.DurationMs)
        logs = append(logs, entry)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(logs)
}

