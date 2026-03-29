// Package apierr defines the canonical JSON error shape returned by every
// handler in the API. All errors must go through here so clients get a
// consistent envelope to parse.
//
// Response shape:
//
//	{
//	  "error": {
//	    "code":    "VEHICLE_NOT_FOUND",   // machine-readable slug
//	    "message": "Vehicle not found.",  // human-readable, safe to display
//	    "details": { ... }                // optional — validation field errors etc.
//	  }
//	}
package apierr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ── Error codes ───────────────────────────────────────────────────────────────
// Keep these in sync with the OpenAPI spec (swaggo annotations on handlers).

const (
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeNotFound       = "NOT_FOUND"
	CodeBadRequest     = "BAD_REQUEST"
	CodeValidation     = "VALIDATION_ERROR"
	CodeConflict       = "CONFLICT"
	CodeInternal       = "INTERNAL_ERROR"
)

// ── Types ─────────────────────────────────────────────────────────────────────

// APIError is the inner error object.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// ErrorResponse is the top-level JSON envelope.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// ValidationDetail maps field names to their error messages.
// Used with CodeValidation responses.
type ValidationDetail map[string]string

// ── Responders ────────────────────────────────────────────────────────────────

func respond(c *gin.Context, status int, code, message string, details any) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error: APIError{Code: code, Message: message, Details: details},
	})
}

func Unauthorized(c *gin.Context, message string) {
	respond(c, http.StatusUnauthorized, CodeUnauthorized, message, nil)
}

func Forbidden(c *gin.Context, message string) {
	respond(c, http.StatusForbidden, CodeForbidden, message, nil)
}

func NotFound(c *gin.Context, message string) {
	respond(c, http.StatusNotFound, CodeNotFound, message, nil)
}

func BadRequest(c *gin.Context, message string) {
	respond(c, http.StatusBadRequest, CodeBadRequest, message, nil)
}

func Validation(c *gin.Context, details ValidationDetail) {
	respond(c, http.StatusUnprocessableEntity, CodeValidation, "Validation failed.", details)
}

func Conflict(c *gin.Context, message string) {
	respond(c, http.StatusConflict, CodeConflict, message, nil)
}

func Internal(c *gin.Context) {
	respond(c, http.StatusInternalServerError, CodeInternal, "An unexpected error occurred.", nil)
}
