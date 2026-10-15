package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"uptime_app/backend/internal/auth"
	"uptime_app/backend/internal/monitor"
)

type monitorService interface {
	Create(context.Context, string, string, int) (monitor.Monitor, error)
	List(context.Context, string) ([]monitor.Monitor, error)
}

func createMonitorHandler(service monitorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			URL             string `json:"url"`
			IntervalSeconds int    `json:"interval_seconds"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			apiError(w, 400, "invalid_input", "Укажите URL и целый интервал в секундах.")
			return
		}
		if decoder.Decode(new(any)) != io.EOF {
			apiError(w, 400, "invalid_input", "Ожидается один JSON-объект.")
			return
		}
		user := r.Context().Value(userContextKey{}).(auth.User)
		value, err := service.Create(r.Context(), user.ID, input.URL, input.IntervalSeconds)
		if errors.Is(err, monitor.ErrInvalid) {
			apiError(w, 400, "invalid_input", "Укажите корректный HTTP/HTTPS URL без логина и пароля и интервал от 5 секунд до 7 дней.")
			return
		}
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, http.StatusCreated, value)
	}
}

func listMonitorsHandler(service monitorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey{}).(auth.User)
		values, err := service.List(r.Context(), user.ID)
		if err != nil {
			serviceError(w, err)
			return
		}
		if values == nil {
			values = []monitor.Monitor{}
		}
		jsonResponse(w, http.StatusOK, values)
	}
}
