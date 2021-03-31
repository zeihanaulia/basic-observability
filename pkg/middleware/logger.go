package http

import (
	"net/http"
	"time"

	mylog "github.com/zeihanaulia/basic-observability/pkg/log"
	"go.uber.org/zap"
)

type LoggerConfig struct {
	SkipURLPath []string
}

func Logger(log mylog.Factory, cfg LoggerConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if len(cfg.SkipURLPath) > 0 {
				if contains(cfg.SkipURLPath, r.URL.Path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			log.Bg().Info(
				"Start Request",
				zap.Any("start_at", time.Now().Format(time.RFC3339)),
				zap.String("method", r.Method),
				zap.String("url", r.URL.Path),
			)

			defer log.Bg().Info(
				"End Request",
				zap.Any("start_at", time.Now().Format(time.RFC3339)),
				zap.String("method", r.Method),
				zap.String("url", r.URL.Path),
			)

			next.ServeHTTP(w, r)
		})
	}
}
