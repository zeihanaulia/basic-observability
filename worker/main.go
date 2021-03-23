package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/nats-io/stan.go"
)

func healthz(w http.ResponseWriter, r *http.Request) {
	fmt.Println(w, "OK")
}

func main() {
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

	// Simple Async Subscriber
	sub, _ := sc.Subscribe("foo", func(m *stan.Msg) {
		fmt.Printf("Received a message: %s\n", string(m.Data))
		log.Println("HERE AA")
	})

	// Unsubscribe
	sub.Unsubscribe()

	// Close connection
	sc.Close()

	fmt.Println("Worker subscribed to 'tasks' for processing requests...")
	fmt.Println("Server listening on port 3131...")

	http.HandleFunc("/healthz", healthz)
	if err := http.ListenAndServe(":3131", nil); err != nil {
		log.Fatal(err)
	}
}
