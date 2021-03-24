package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/hashicorp/go.net/context"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	mylog "github.com/zeihanaulia/go-async-request/pkg/log"
	mymdlwr "github.com/zeihanaulia/go-async-request/pkg/middleware"
	tracing "github.com/zeihanaulia/go-async-request/pkg/tracer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/resty.v1"
	"syreclabs.com/go/faker"
)

func main() {
	tracer, closer := tracing.Init()
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	logger, _ := zap.NewDevelopment(
		zap.AddStacktrace(zapcore.FatalLevel),
		zap.AddCallerSkip(1),
	)
	zapLogger := logger.With(zap.String("service", "payment"))
	loggers := mylog.NewFactory(zapLogger)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(mymdlwr.Trace(tracer, mymdlwr.TraceConfig{
		SkipURLPath: []string{
			"/metrics",
		},
	}))
	r.Use(mymdlwr.Metrics("payment_service", []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}...))

	s := Service{tracer, loggers}
	r.Get("/", s.index)
	r.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(":3031", r)
}

type Service struct {
	tracer opentracing.Tracer
	log    mylog.Factory
}

func (s *Service) index(w http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContext(r.Context(), "index")
	defer span.Finish()

	soNo := s.SentOrder(ctx, Payload())

	render.JSON(w, r, Response{soNo})
}

func (s *Service) SentOrder(ctx context.Context, payload RegisterPayment) string {
	span, _ := opentracing.StartSpanFromContext(ctx, "index")
	defer span.Finish()

	url := "http://order:3030/purchase-order"
	headers := http.Header{
		"Content-Type": []string{"application/json"},
	}

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "POST")
	_ = span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(headers),
	)

	var header = make(map[string]string)
	for name, value := range headers {
		header[name] = value[0]
	}

	client := resty.New()
	resp, err := client.R().
		SetHeaders(header).
		SetBody(payload).
		Post(url)
	if err != nil {
		log.Println(err)
	}

	res := Response{}
	_ = json.Unmarshal(resp.Body(), &res)

	return res.SONumber
}

type Response struct {
	SONumber string `json:"so_number"`
}

func Payload() RegisterPayment {
	return RegisterPayment{
		TrxID: fmt.Sprintf("PAY-%d", faker.Number().NumberInt32(8)),
		Customer: Customer{
			ID:      fmt.Sprintf("WP%d", faker.Number().NumberInt32(8)),
			Name:    faker.Name().Name(),
			Address: faker.Address().String(),
		},
		Order: []Order{
			{
				StoreID: faker.Number().NumberInt(2),
				Items:   Items(),
			},
		},
	}
}

func Items() []Item {
	n := faker.Number().NumberInt(1)
	resp := make([]Item, 0)
	for i := 0; i < n; i++ {
		resp = append(resp, Item{
			ProductID: faker.Number().NumberInt(5),
			SKU:       faker.NumerifyAndLetterify("???###"),
			Name:      faker.Commerce().ProductName(),
			Uom:       faker.RandomChoice([]string{"Dus", "Bungkus", "Slop", "Botol", "Sachet"}),
			Price:     float64(faker.Commerce().Price()),
			Quantity:  faker.Number().NumberInt(2),
		})
	}
	return resp
}
