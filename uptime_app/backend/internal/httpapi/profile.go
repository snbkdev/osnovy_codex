package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"uptime_app/backend/internal/auth"
)

func profileHandler(service authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			FullName    string  `json:"full_name"`
			DateOfBirth *string `json:"date_of_birth"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			apiError(w, 400, "invalid_input", "Проверьте данные профиля.")
			return
		}
		if decoder.Decode(new(any)) != io.EOF {
			apiError(w, 400, "invalid_input", "Ожидается один JSON-объект.")
			return
		}
		user, err := service.UpdateProfile(r.Context(), strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), input.FullName, input.DateOfBirth)
		if errors.Is(err, auth.ErrInvalid) {
			apiError(w, 400, "invalid_input", "ФИО должно содержать хотя бы одну букву и не более 200 символов. Дата рождения должна быть корректной и не позднее сегодняшней.")
			return
		}
		if err != nil {
			serviceError(w, err)
			return
		}
		jsonResponse(w, 200, dto(user))
	}
}
