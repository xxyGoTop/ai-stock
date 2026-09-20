package response

import (
	"encoding/json"
	"net/http"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func JSON(w http.ResponseWriter, status int, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Body{Code: code, Message: message, Data: data})
}

func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, 0, "ok", data)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, status, message, nil)
}
