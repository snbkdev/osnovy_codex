package httpapi

import (
	"errors"
	"net/http"
	"uptime_app/backend/internal/auth"
)

// @Summary Update profile
// @Tags Profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body profileInput true "Request body"
// @Success 200 {object} userDTO
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 429 {object} errorResponse
// @Failure 503 {object} errorResponse
// @Router /auth/profile [post]
func profileHandler(service authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input profileInput
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
