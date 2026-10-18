package httpapi

// Named HTTP payloads are shared by handlers and generated API documentation.
type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
type messageResponse struct {
	Message string `json:"message"`
}
type healthResponse struct {
	Status string `json:"status"`
}
type tokenResponse struct {
	AccessToken string  `json:"access_token"`
	TokenType   string  `json:"token_type"`
	ExpiresIn   int     `json:"expires_in"`
	User        userDTO `json:"user"`
}
type credentialsInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type emailInput struct {
	Email string `json:"email" binding:"required"`
}
type verificationInput struct {
	Token string `json:"token" binding:"required"`
}
type resetInput struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type profileInput struct {
	FullName    string  `json:"full_name"`
	DateOfBirth *string `json:"date_of_birth" extensions:"x-nullable"`
}
type monitorInput struct {
	URL             string `json:"url" binding:"required"`
	IntervalSeconds int    `json:"interval_seconds" minimum:"5" maximum:"604800" binding:"required"`
}
