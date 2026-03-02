package utils

import "net/http"

func Error(w http.ResponseWriter, code int, msg string) {
	JSON(w, code, map[string]string{
		"error": msg,
	})
}
