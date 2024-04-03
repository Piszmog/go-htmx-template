package middleware

import "net/http"

type Handler func(http.Handler) http.Handler

func Chain(handlers ...Handler) Handler {
	if len(handlers) == 0 {
		return defaultHandler
	}

	return func(next http.Handler) http.Handler {
		for i := len(handlers) - 1; i >= 0; i-- {
			next = handlers[i](next)
		}
		return next
	}
}

func defaultHandler(next http.Handler) http.Handler {
	return next
}
