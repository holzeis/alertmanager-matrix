package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "time"
)

// ---- Config ----

var (
    matrixHomeserver = os.Getenv("MATRIX_HOMESERVER") // e.g. https://matrix.org
    matrixToken      = os.Getenv("MATRIX_TOKEN")
    matrixRoomID     = os.Getenv("MATRIX_ROOM_ID")
    listenAddr       = ":8080"
)

// ---- Alertmanager structures ----

type AlertmanagerWebhook struct {
    Status string  `json:"status"`
    Alerts []Alert `json:"alerts"`
}

type Alert struct {
    Status      string            `json:"status"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    StartsAt    string            `json:"startsAt"`
    EndsAt      string            `json:"endsAt"`
}

// ---- Matrix payload ----

type MatrixMessage struct {
    MsgType string `json:"msgtype"`
    Body    string `json:"body"`
}

// ---- Handlers ----

func alertHandler(w http.ResponseWriter, r *http.Request) {
    var webhook AlertmanagerWebhook

    if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
       http.Error(w, err.Error(), http.StatusBadRequest)
       return
    }

    for _, alert := range webhook.Alerts {
       message := formatAlertBySeverity(alert)
       if err := sendToMatrix(message); err != nil {
          log.Println("Matrix send failed:", err)
       }
    }

    w.WriteHeader(http.StatusOK)
}

func watchdogHandler(w http.ResponseWriter, r *http.Request) {
    message := "✅ Daily Watchdog: All systems operational!"
    if err := sendToMatrix(message); err != nil {
        log.Println("Matrix send failed:", err)
        http.Error(w, "Failed to send watchdog message", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
}

// ---- Alert Formatting ----

func formatAlertBySeverity(alert Alert) string {
    name := alert.Labels["alertname"]
    severity := alert.Labels["severity"]
    summary := alert.Annotations["summary"]
    description := alert.Annotations["description"]

    if alert.Status == "resolved" {
        return fmt.Sprintf("✅ Resolved Alert: %s", name)
    }

    switch severity {
    case "critical":
        return fmt.Sprintf(
            "🚨 %s\n%s\n\n%s",
            name,
            summary,
            description,
        )
    case "warning":
        return fmt.Sprintf(
            "⚠️ %s\n%s\n\n%s",
            name,
            summary,
            description,
        )
    default:
        return fmt.Sprintf(
            "ℹ️ %s\n%s\n\n%s",
            name,
            summary,
            description,
        )
    }
}

// ---- Matrix sender ----

func sendToMatrix(message string) error {
    if matrixHomeserver == "" {
        return fmt.Errorf("MATRIX_HOMESERVER not set")
    }
    if matrixToken == "" {
        return fmt.Errorf("MATRIX_TOKEN not set")
    }
    if matrixRoomID == "" {
        return fmt.Errorf("MATRIX_ROOM_ID not set")
    }

    encodedRoomID := url.PathEscape(matrixRoomID)
    txnID := fmt.Sprintf("txn-%d", time.Now().UnixNano())
    apiURL := fmt.Sprintf(
        "%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s?access_token=%s",
        matrixHomeserver,
        encodedRoomID,
        txnID,
        matrixToken,
    )

    payload := MatrixMessage{
        MsgType: "m.text",
        Body:    message,
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal payload: %w", err)
    }

    req, err := http.NewRequest("PUT", apiURL, bytes.NewBuffer(body))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 300 {
        return fmt.Errorf("matrix returned status %s", resp.Status)
    }

    return nil
}

// ---- Main ----

func main() {
    // Validate required environment variables
    if matrixHomeserver == "" || matrixToken == "" || matrixRoomID == "" {
        log.Fatal("Required environment variables not set: MATRIX_HOMESERVER, MATRIX_TOKEN, MATRIX_ROOM_ID")
    }

    http.HandleFunc("/alerts", alertHandler)
    http.HandleFunc("/watchdog", watchdogHandler)

    log.Println("Listening on", listenAddr)
    log.Fatal(http.ListenAndServe(listenAddr, nil))
}