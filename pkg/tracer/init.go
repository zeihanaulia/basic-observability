package tracer

import (
	"io"
	"log"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

func Init() (tracer opentracing.Tracer, closer io.Closer) {
	cfg, err := config.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Fatalf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	tracer, closer, err = cfg.NewTracer()
	if err != nil {
		log.Fatalf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}

	return
}
