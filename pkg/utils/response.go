package utils

// Response - Standart javob strukturasi
type Response struct {
	Success bool        `json:"success"`         // Muvaffaqiyatlimi
	Message string      `json:"message"`         // Xabar
	Data    interface{} `json:"data,omitempty"`  // Ma'lumotlar
	Error   string      `json:"error,omitempty"` // Xatolik
}

// SuccessResponse - Muvaffaqiyatli javob
func SuccessResponse(message string, data interface{}) *Response {
	return &Response{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// ErrorResponse - Xatolik javobi
func ErrorResponse(message string, err error) *Response {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	return &Response{
		Success: false,
		Message: message,
		Error:   errorMsg,
	}
}
