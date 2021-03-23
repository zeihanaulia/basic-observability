package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	stan "github.com/nats-io/stan.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

func main() {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	opentracing.SetGlobalTracer(tracer)

	sc, err := stan.Connect(
		"test-cluster",
		"order-test",
		stan.NatsURL("nats://nats:4222"),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatalf("Nats Connection lost, reason: %v", reason)
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	s := Service{sc, tracer}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", s.index)
	_ = http.ListenAndServe(":3030", r)
}

type Service struct {
	stan   stan.Conn
	tracer opentracing.Tracer
}

type TraceMsg struct {
	bytes.Buffer
}

var t TraceMsg

func (s Service) index(w http.ResponseWriter, r *http.Request) {
	sub := "bar"

	pubSpan := s.tracer.StartSpan("Published Message", ext.SpanKindProducer)
	ext.MessageBusDestination.Set(pubSpan, sub)
	defer pubSpan.Finish()

	carier := map[string]string{}
	_ = s.tracer.Inject(pubSpan.Context(), opentracing.TextMap, opentracing.TextMapCarrier(carier))

	payload := Payload(carier["uber-trace-id"])
	jsonString, _ := json.Marshal(payload)
	msg := []byte(jsonString)

	// Simple Synchronous Publisher
	// _ = s.stan.Publish(sub, msg) // does not return until an ack has been received from NATS Streaming
	_, _ = s.stan.PublishAsync(sub, msg, stan.AckHandler(func(s string, err error) {
		fmt.Println(s, err)
	}))

	_, _ = w.Write([]byte("welcome"))
}

func Payload(uberTraceID string) OrderPayload {
	return OrderPayload{
		UberTraceID:  uberTraceID,
		SONumber:     "MBS-12312-123123",
		PaymentTrxID: "PMT-123123-9876",
		StoreID:      10,
		Items: []Item{
			{
				ProductID: 1,
				SKU:       "ABC001",
				Uom:       "Dus",
				UomID:     1,
				Price:     100000,
				Quantity:  10,
			},
		},
	}
}

type OrderPayload struct {
	UberTraceID  string `json:"uber_trace_id,omitempty"`
	SONumber     string `json:"so_number,omitempty"`
	PaymentTrxID string `json:"payment_trx_id,omitempty"`
	StoreID      int    `json:"store_id,omitempty"`
	Items        []Item `json:"items,omitempty"`
}

type Item struct {
	ProductID int     `json:"product_id,omitempty"`
	SKU       string  `json:"sku,omitempty"`
	Uom       string  `json:"uom,omitempty"`
	UomID     int     `json:"uom_id,omitempty"`
	Price     float64 `json:"price,omitempty"`
	Quantity  float64 `json:"quantity,omitempty"`
}
