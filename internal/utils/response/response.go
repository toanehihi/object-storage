package response

type ApiResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
}

func OK[T any](data T, message string) ApiResponse[T] {
	return ApiResponse[T]{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func Fail[T any](message string) ApiResponse[T] {
	return ApiResponse[T]{
		Success: false,
		Message: message,
	}
}
