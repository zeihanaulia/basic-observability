package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
	r.Use(middleware.Logger)
	r.Use(mymdlwr.Trace(tracer, mymdlwr.TraceConfig{
		SkipURLPath: []string{
			"/metrics",
		},
	}))
	r.Use(mymdlwr.Metrics("order_service"))

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
