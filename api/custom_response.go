package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/GitDataAI/jiaozifs/auth"
	"github.com/GitDataAI/jiaozifs/models"
)

type JiaozifsResponse struct {
	http.ResponseWriter
}

// JSON convert object to json format and write to response,
// if not specific code, default code is 200. given code will
// overwrite default code, if more than one code, the first one will be used.
func (response *JiaozifsResponse) JSON(v any, code ...int) {
	response.Header().Set("Content-Type", "application/json")

	if len(code) == 0 {
		response.WriteHeader(http.StatusOK)
	} else {
		response.WriteHeader(code[0])
	}

	err := json.NewEncoder(response.ResponseWriter).Encode(v)
	if err != nil {
		response.Error(err)
		return
	}
}

// OK response with 200
func (response *JiaozifsResponse) OK() {
	response.WriteHeader(http.StatusOK)
}

// NotFound response with 404
func (response *JiaozifsResponse) NotFound() {
	response.WriteHeader(http.StatusNotFound)
}
func (response *JiaozifsResponse) Forbidden() {
	response.WriteHeader(http.StatusForbidden)
}

// Unauthorized response with 401
func (response *JiaozifsResponse) Unauthorized() {
	response.WriteHeader(http.StatusUnauthorized)
}

func (response *JiaozifsResponse) BadRequest(msg string, args ...any) {
	response.WriteHeader(http.StatusBadRequest)
	_, _ = response.Write([]byte(fmt.Sprintf(msg, args...)))
}

// Error response with 500 and error message
func (response *JiaozifsResponse) Error(err error) {
	if errors.Is(err, models.ErrNotFound) {
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write([]byte(err.Error()))
		return
	}
	if errors.Is(err, auth.ErrUserNotFound) {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}

	var codeErr ErrCode
	if errors.As(err, &codeErr) {
		response.WriteHeader(int(codeErr))
		_, _ = response.Write([]byte(err.Error()))
		return
	}

	response.WriteHeader(http.StatusInternalServerError)
	_, _ = response.Write([]byte(err.Error()))
}

// String response and string
// if not specific code, default code is 200. given code will
// overwrite default code, if more than one code, the first one will be used.
func (response *JiaozifsResponse) String(msg string, code ...int) {
	response.Header().Set("Content-Type", "text/plain;charset=UTF-8")

	if len(code) == 0 {
		response.WriteHeader(http.StatusOK)
	} else {
		response.WriteHeader(code[0])
	}
	_, _ = response.Write([]byte(msg))
}

// Code response with uncommon code
func (response *JiaozifsResponse) Code(code int) {
	response.WriteHeader(code)
}

type ErrCode int

func NewErrCode(code int) ErrCode {
	return ErrCode(code)
}
func (err ErrCode) Error() string {
	return fmt.Sprintf("code %d msg %s", err, http.StatusText(int(err)))
}
