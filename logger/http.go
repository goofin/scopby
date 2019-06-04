package logger

import (
	"net/http"

	"go.uber.org/zap"
)

func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		Logger.Info("http",
			zap.String("method", req.Method),
			zap.String("url", req.URL.String()),
			zap.String("remote-addr", req.RemoteAddr))

		h.ServeHTTP(w, req)
	})
}
