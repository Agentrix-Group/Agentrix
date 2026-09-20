package common

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

type Response struct {
	HttpStatusCode int    `json:"httpStatusCode"`
	ErrorCode      int    `json:"errorCode,omitempty"`
	Message        string `json:"message"`
	RequestId      string `json:"requestId,omitempty"`
	Diagnostic     string `json:"diagnostic,omitempty"`
}

func writeJSON(w http.ResponseWriter, httpStatus int, data any) {
	w.Header().Set("Content-Type", ContentTypeJSON)
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(data)
}

func WriteErrorResponse(w http.ResponseWriter, errorCode int) {
	WriteErrorMessage(w, errorCode, ErrorMessages[errorCode])
}

func WriteErrorMessage(w http.ResponseWriter, errorCode int, message string) {
	httpStatus := ErrorCodes[errorCode]
	if httpStatus == 0 {
		httpStatus = http.StatusInternalServerError
	}
	writeJSON(w, httpStatus, Response{
		HttpStatusCode: httpStatus,
		ErrorCode:      errorCode,
		Message:        message,
	})
}

func WriteDiagnosticError(w http.ResponseWriter, ctx context.Context, errorCode int, message string, diagnostic string) {
	reqID := tracer.RequestID(ctx)
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "dev"
	}
	isDev := mode == "dev"

	diag := ""
	if isDev {
		diag = diagnostic
	}

	httpStatus := ErrorCodes[errorCode]
	if httpStatus == 0 {
		httpStatus = http.StatusInternalServerError
	}
	writeJSON(w, httpStatus, Response{
		HttpStatusCode: httpStatus,
		ErrorCode:      errorCode,
		Message:        message,
		RequestId:      reqID,
		Diagnostic:     diag,
	})
}

func WriteSuccessResponse(w http.ResponseWriter, httpStatus int, message string) {
	writeJSON(w, httpStatus, Response{
		HttpStatusCode: httpStatus,
		Message:        message,
	})
}

func WriteObjectResponse(w http.ResponseWriter, httpStatus int, data any) {
	writeJSON(w, httpStatus, data)
}
