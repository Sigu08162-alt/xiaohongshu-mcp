package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"log/slog"
)

// respondError 返回错误响应
func respondError(c *gin.Context, statusCode int, code, message string, details any) {
	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	slog.Error("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), statusCode)

	c.JSON(statusCode, response)
}

// respondSuccess 返回成功响应，并自动设置 account 标识用于日志
func respondSuccess(c *gin.Context, data any, message string) {
	c.Set("account", "ai-report")

	response := SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	}

	slog.Info("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), http.StatusOK)

	c.JSON(http.StatusOK, response)
}

// bindJSON binds a JSON request body and responds with a 400 error on failure.
// Returns true if binding succeeded, false if an error response was already sent.
func bindJSON(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "请求参数错误", err.Error())
		return false
	}
	return true
}
