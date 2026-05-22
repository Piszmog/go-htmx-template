package middleware

import (
	"net/http"
	"slices"
)

type Handler func(http.Handler) http.Handler

func Chain(handlers ...Handler) Handler {
	if len(handlers) == 0 {
		return defaultHandler
	}

	return func(next http.Handler) http.Handler {
		for _, h := range slices.Backward(handlers) {
			next = h(next)
		}
		return next
	}
}

func defaultHandler(next http.Handler) http.Handler {
	return next
}
