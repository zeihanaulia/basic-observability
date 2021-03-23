package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/nats-io/stan.go"
	"github.com/nats-io/stan.go/pb"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	mylog "github.com/zeihanaulia/go-async-request/pkg/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func healthz(w http.ResponseWriter, r *http.Request) {
	fmt.Println(w, "OK")
}

type TraceMsg struct {
	bytes.Buffer
}

func NewTraceMsg(m *stan.Msg) *TraceMsg {
	b := bytes.NewBuffer(m.Data)
	return &TraceMsg{*b}
}

func main() {

	logger, _ := zap.NewDevelopment(
		zap.AddStacktrace(zapcore.FatalLevel),
		zap.AddCallerSkip(1),
	)
	zapLogger := logger.With(zap.String("service", "payment"))
	loggers := mylog.NewFactory(zapLogger)

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
		"worker-test",
		stan.NatsURL("nats://nats:4222"),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatalf("Nats Connection lost, reason: %v", reason)
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Simple Async Subscriber
	unsubscribe := false
	durable := "order-worker"
	startOpt := stan.StartAt(pb.StartPosition_NewOnly)
	subj, i := os.Getenv("NATS_SUBJECT"), 0
	mcb := func(msg *stan.Msg) {
		i++

		payload := OrderPayload{}
		_ = json.Unmarshal(msg.Data, &payload)

		operationName := "Received Message"
		ctx := context.Background()
		span, ctx := SpanFromTraceID(ctx, operationName, msg.Subject, payload.UberTraceID)
		defer span.Finish()

		loggers.For(ctx).Info("Worker Receiver Order", zap.String("message", string(msg.Data)))

		printMsg(msg, i)
	}
	sub, err := sc.QueueSubscribe(subj, "qg-order", mcb, startOpt, stan.DurableName(durable))
	if err != nil {
		sc.Close()
		log.Fatal(err)
	}

	fmt.Println("Worker subscribed to 'tasks' for processing requests...")
	fmt.Println("Server listening on port 3131...")

	signalChan := make(chan os.Signal, 1)
	cleanupDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			fmt.Printf("\nReceived an interrupt, unsubscribing and closing connection...\n\n")
			// Do not unsubscribe a durable on exit, except if asked to.
			if durable == "" || unsubscribe {
				sub.Unsubscribe()
			}
			sc.Close()
			cleanupDone <- true
		}
	}()
	<-cleanupDone
}

func SpanFromTraceID(ctx context.Context, operationName, subject, uberTraceID string) (opentracing.Span, context.Context) {
	carier := map[string]string{"uber-trace-id": uberTraceID}

	span, ctx := opentracing.StartSpanFromContext(ctx, operationName)
	sc, err := opentracing.GlobalTracer().Extract(opentracing.TextMap, opentracing.TextMapCarrier(carier))
	if err == nil {
		span = opentracing.StartSpan(operationName, ext.SpanKindConsumer, opentracing.FollowsFrom(sc))
		ext.MessageBusDestination.Set(span, subject)
	}

	return span, ctx
}

func printMsg(m *stan.Msg, i int) {
	log.Printf("[#%d] Received: %s\n", i, m)
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
