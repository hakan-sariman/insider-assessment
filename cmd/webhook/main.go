package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type response struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}

type inbound struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	id := uuid.New().String()

	limited := http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	dec := json.NewDecoder(limited)
	dec.DisallowUnknownFields()
	var in inbound
	var decodeErr string
	if err := dec.Decode(&in); err != nil {
		decodeErr = err.Error()
	}
	headersMap := make(map[string]string, len(r.Header))
	for name, values := range r.Header {
		headersMap[name] = strings.Join(values, ", ")
	}
	logEntry := struct {
		Method      string            `json:"method"`
		URL         string            `json:"url"`
		Headers     map[string]string `json:"headers"`
		To          string            `json:"to"`
		Content     string            `json:"content"`
		DecodeError string            `json:"decodeError,omitempty"`
	}{
		Method:      r.Method,
		URL:         r.URL.String(),
		Headers:     headersMap,
		To:          strings.TrimSpace(in.To),
		Content:     strings.TrimSpace(in.Content),
		DecodeError: strings.TrimSpace(decodeErr),
	}
	if b, err := json.Marshal(logEntry); err == nil {
		log.Println(string(b))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(response{Message: "Accepted", MessageID: id})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	srv := &http.Server{
		Addr:         ":8090",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("webhook server listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
