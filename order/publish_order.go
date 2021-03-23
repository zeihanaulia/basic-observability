package main

import (
	"context"
	"encoding/json"
	"fmt"

	stan "github.com/nats-io/stan.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

func (s *Service) PublishOrder(ctx context.Context, payload RegisterPayment) {
	sub := "purchase.order"
	span, _ := opentracing.StartSpanFromContext(ctx, "Publish Order", ext.SpanKindProducer)
	ext.MessageBusDestination.Set(span, sub)
	defer span.Finish()

	carier := map[string]string{}
	_ = s.tracer.Inject(span.Context(), opentracing.TextMap, opentracing.TextMapCarrier(carier))

	payload = Payload(carier["uber-trace-id"], payload)
	jsonString, _ := json.Marshal(payload)
	msg := []byte(jsonString)

	_, _ = s.stan.PublishAsync(sub, msg, stan.AckHandler(func(s string, err error) {
		fmt.Println(s, err)
	}))

}

func Payload(uberTraceID string, payload RegisterPayment) RegisterPayment {
	payload.UberTraceID = uberTraceID
	return payload
}
