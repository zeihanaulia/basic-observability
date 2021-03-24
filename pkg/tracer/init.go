package tracer

import (
	"io"
	"log"
	"os"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics"
	"github.com/uber/jaeger-lib/metrics/prometheus"
)

func Init() (tracer opentracing.Tracer, closer io.Closer) {
	cfg, err := config.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Fatalf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}
	metricsFactory := prometheus.New().Namespace(metrics.NSOptions{Name: os.Getenv("JAEGER_SERVICE_NAME"), Tags: nil})
	tracer, closer, err = cfg.NewTracer(
		config.Metrics(metricsFactory),
		config.Logger(jaeger.StdLogger),
	)
	if err != nil {
		log.Fatalf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}

	return
}
