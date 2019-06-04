package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/zeebo/errs"
)

var Err = errs.Class("rest")

func jsonEncode(w io.Writer, data interface{}) {
	// TODO(jeff): what if this errors?
	buf, _ := json.MarshalIndent(data, "", "\t")
	w.Write(buf)
}

func handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	jsonEncode(w, map[string]interface{}{
		"status":  "error",
		"error":   err.Error(),
		"verbose": fmt.Sprintf("%+v", err),
	})
}

func handleSuccess(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	jsonEncode(w, map[string]interface{}{
		"status": "ok",
		"data":   data,
	})
}

// TODO: This sucks but we return a concretr func because alien is garbage.

func wrapHandler(fn func(context.Context, *http.Request) (interface{}, error)) (
	out func(http.ResponseWriter, *http.Request)) {

	return func(w http.ResponseWriter, req *http.Request) {
		data, err := fn(req.Context(), req)
		if err != nil {
			handleError(w, err)
		} else {
			handleSuccess(w, data)
		}
	}
}
