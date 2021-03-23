package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	stan "github.com/nats-io/stan.go"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", index)
	_ = http.ListenAndServe(":3030", r)
}

func index(w http.ResponseWriter, r *http.Request) {
	sc, err := stan.Connect(
		"test-cluster",
		"test",
		stan.NatsURL("nats://nats:4222"),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatalf("Nats Connection lost, reason: %v", reason)
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Simple Synchronous Publisher
	_ = sc.Publish("foo", []byte("Hello World")) // does not return until an ack has been received from NATS Streaming

	// Close connection
	sc.Close()

	log.Println("HERE")
	_, _ = w.Write([]byte("welcome"))
}
