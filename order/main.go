package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	stan "github.com/nats-io/stan.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	mylog "github.com/zeihanaulia/go-async-request/pkg/log"
	mymdlwr "github.com/zeihanaulia/go-async-request/pkg/middleware"
	tracing "github.com/zeihanaulia/go-async-request/pkg/tracer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	tracer, closer := tracing.Init()
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

	logger, _ := zap.NewDevelopment(
		zap.AddStacktrace(zapcore.FatalLevel),
		zap.AddCallerSkip(1),
	)
	zapLogger := logger.With(zap.String("service", "customer"))
	loggers := mylog.NewFactory(zapLogger)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(mymdlwr.Trace(tracer, mymdlwr.TraceConfig{
		SkipURLPath: []string{
			"/metrics",
		},
	}))
	r.Use(mymdlwr.Metrics("order_service"))

	s := Service{sc, tracer, loggers}
	r.Get("/", s.index)
	r.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(":3030", r)
}

type Service struct {
	stan   stan.Conn
	tracer opentracing.Tracer
	log    mylog.Factory
}

func (s *Service) index(w http.ResponseWriter, r *http.Request) {
	sub := "bar"
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "index", ext.SpanKindProducer)
	ext.MessageBusDestination.Set(span, sub)
	defer span.Finish()

	s.log.For(ctx).Info("Test info", zap.String("test", "info"))

	carier := map[string]string{}
	_ = s.tracer.Inject(span.Context(), opentracing.TextMap, opentracing.TextMapCarrier(carier))

	payload := Payload(carier["uber-trace-id"])
	jsonString, _ := json.Marshal(payload)
	msg := []byte(jsonString)

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
