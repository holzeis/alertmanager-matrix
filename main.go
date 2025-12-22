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
		message := formatAlert(alert)
		if err := sendToMatrix(message); err != nil {
			log.Println("Matrix send failed:", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func formatAlert(alert Alert) string {
	name := alert.Labels["alertname"]
	severity := alert.Labels["severity"]
	summary := alert.Annotations["summary"]
	description := alert.Annotations["description"]

	return fmt.Sprintf(
		"🚨 **Alert:** %s\n**Severity:** %s\n**Status:** %s\n\n%s\n%s",
		name,
		severity,
		alert.Status,
		summary,
		description,
	)
}

// ---- Matrix sender ----

func sendToMatrix(message string) error {
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

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("PUT", apiURL, bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 300 {
        return fmt.Errorf("matrix returned status %s", resp.Status)
    }

    return nil
}

// ---- Main ----

func main() {
	http.HandleFunc("/alerts", alertHandler)

	log.Println("Listening on", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
