package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/render"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"go.uber.org/zap"
	"syreclabs.com/go/faker"
)

func (s *Service) purchaseOrder(w http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "purchaseOrder", ext.SpanKindProducer)
	defer span.Finish()

	decoder := json.NewDecoder(r.Body)
	var request RegisterPayment
	if err := decoder.Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, err)
		return
	}

	b, _ := json.Marshal(request)
	s.log.For(ctx).Info("Receive Request", zap.String("payload", string(b)))
	s.log.Bg().Info("test log")

	soNumber := fmt.Sprintf("SO%d", faker.Number().NumberInt(5))
	request.SONumber = soNumber

	s.PublishOrder(ctx, request)

	render.JSON(w, r, Response{SONumber: soNumber})
}

type Response struct {
	SONumber string `json:"so_number"`
}
