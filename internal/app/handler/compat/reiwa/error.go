package reiwa

import "net/http"

type ReiwaErrorResponse struct {
	Error ReiwaError `json:"error"`
}

type ReiwaError struct {
	Code              int    `json:"code"`
	Message           string `json:"message"`
	AdditionalMessage string `json:"additional_message"`
}

func NewReiwaErrorResponse(statusCode int, additionalMessage string) ReiwaErrorResponse {
	return ReiwaErrorResponse{
		Error: ReiwaError{
			Code:              statusCode,
			Message:           getMessageForStatusCode(statusCode),
			AdditionalMessage: additionalMessage,
		},
	}
}

func getMessageForStatusCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "bad request."
	case http.StatusNotFound:
		return "not found."
	case http.StatusMethodNotAllowed:
		return "the requested method is not allowed."
	case http.StatusTooManyRequests:
		return "too many requests."
	case http.StatusServiceUnavailable:
		return "service unavailable."
	default:
		return "service unavailable."
	}
}

func NewBadRequestError(additionalMessage string) ReiwaErrorResponse {
	return NewReiwaErrorResponse(http.StatusBadRequest, additionalMessage)
}

func NewNotFoundError(additionalMessage string) ReiwaErrorResponse {
	return NewReiwaErrorResponse(http.StatusNotFound, additionalMessage)
}

func NewMethodNotAllowedError(additionalMessage string) ReiwaErrorResponse {
	return NewReiwaErrorResponse(http.StatusMethodNotAllowed, additionalMessage)
}

func NewTooManyRequestsError(additionalMessage string) ReiwaErrorResponse {
	return NewReiwaErrorResponse(http.StatusTooManyRequests, additionalMessage)
}

func NewServiceUnavailableError(additionalMessage string) ReiwaErrorResponse {
	return NewReiwaErrorResponse(http.StatusServiceUnavailable, additionalMessage)
}
