package httpapi

import (
	"context"
	"errors"
	"net/http"

	"uptime_app/backend/internal/monitor"
)

type monitorService interface {
	Create(context.Context, string, string, int) (monitor.Monitor, error)
	List(context.Context, string, ...string) ([]monitor.Monitor, error)
}

// @Summary Create monitor
// @Tags Monitors
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body monitorInput true "Request body"
// @Success 201 {object} monitor.Monitor
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 503 {object} errorResponse
// @Router /monitors [post]
func createMonitorHandler(service monitorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input monitorInput
		if !jsonInput(w, r, 4096, &input, "Укажите URL и целый интервал в секундах.") {
			return
		}
		user := currentIdentity(r).User
		value, err := service.Create(r.Context(), user.ID, input.URL, input.IntervalSeconds)
		if errors.Is(err, monitor.ErrQuota) {
			apiError(w, 409, "monitor_quota", "Достигнут лимит в 100 мониторов.")
			return
		}
		if errors.Is(err, monitor.ErrInvalid) {
			apiError(w, 400, "invalid_input", "Укажите корректный HTTP/HTTPS URL без логина и пароля и интервал от 5 секунд до 7 дней.")
			return
		}
		if errors.Is(err, monitor.ErrUnauthorized) {
			apiError(w, 401, "unauthorized", "Войдите в аккаунт.")
			return
		}
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, http.StatusCreated, value)
	}
}

// @Summary List monitors
// @Tags Monitors
// @Produce json
// @Security BearerAuth
// @Param before query string false "Pagination cursor"
// @Success 200 {array} monitor.Monitor
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 503 {object} errorResponse
// @Router /monitors [get]
func listMonitorsHandler(service monitorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := currentIdentity(r).User
		values, err := service.List(r.Context(), user.ID, r.URL.Query().Get("before"))
		if errors.Is(err, monitor.ErrInvalid) {
			apiError(w, 400, "invalid_input", "Некорректный курсор.")
			return
		}
		if errors.Is(err, monitor.ErrUnauthorized) {
			apiError(w, 401, "unauthorized", "Войдите в аккаунт.")
			return
		}
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
