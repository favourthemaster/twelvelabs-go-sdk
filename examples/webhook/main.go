// Command webhook runs an HTTP server that verifies TwelveLabs webhook
// signatures.
//
// Usage:
//
//	WEBHOOK_SECRET=... go run ./examples/webhook
package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/favourthemaster/twelvelabs-go-sdk"
)

func main() {
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("set WEBHOOK_SECRET to the signing secret from the TwelveLabs console")
	}
	verifier := twelvelabs.NewWebhookVerifier(secret)

	http.HandleFunc("/webhooks/twelvelabs", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		event, err := verifier.VerifyAndParse(body, r.Header.Get("TL-Signature"))
		if err != nil {
			log.Printf("rejected webhook: %v", err)
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		switch event.Type {
		case twelvelabs.WebhookEventIndexTaskReady, twelvelabs.WebhookEventIndexTaskFailed:
			data, err := event.TaskData()
			if err != nil {
				log.Printf("bad payload: %v", err)
				break
			}
			log.Printf("%s task=%s status=%s", event.Type, data.ID, data.Status)
		default:
			log.Printf("unhandled event %s", event.Type)
		}

		w.WriteHeader(http.StatusOK)
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
