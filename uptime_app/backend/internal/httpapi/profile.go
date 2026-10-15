package httpapi

import (
	"errors"
	"net/http"
	"uptime_app/backend/internal/auth"
)

func profileHandler(service authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			FullName    string  `json:"full_name"`
			DateOfBirth *string `json:"date_of_birth"`
		}
		if !jsonInput(w, r, 16*1024, &input, "Проверьте данные профиля.") {
			return
		}
		user, err := service.UpdateProfile(r.Context(), currentIdentity(r), input.FullName, input.DateOfBirth)
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
