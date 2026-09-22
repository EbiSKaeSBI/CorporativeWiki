package dto

// ErrorResponse представляет стандартный ответ с ошибкой
// @Description Стандартный ответ при ошибке
type ErrorResponse struct {
	Error string `json:"error" example:"Описание ошибки"`
}