package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	stan "github.com/nats-io/stan.go"
	"github.com/opentracing/opentracing-go"
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
	zapLogger := logger.With(zap.String("service", "order"))
	loggers := mylog.NewFactory(zapLogger)

	r := chi.NewRouter()
	r.Use(mymdlwr.Logger(loggers, mymdlwr.LoggerConfig{
		SkipURLPath: []string{
			"/metrics",
		},
	}))
	r.Use(mymdlwr.Trace(tracer, mymdlwr.TraceConfig{
		SkipURLPath: []string{
			"/metrics",
		},
	}))
	r.Use(mymdlwr.Metrics("order_service", []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}...))

	s := Service{sc, tracer, loggers}
	r.Get("/", s.index)
	r.Post("/purchase-order", s.purchaseOrder)
	r.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(":3030", r)
}

type Service struct {
	stan   stan.Conn
	tracer opentracing.Tracer
	log    mylog.Factory
}
