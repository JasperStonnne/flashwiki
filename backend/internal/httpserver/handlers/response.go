package handlers

import "github.com/gin-gonic/gin"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Envelope struct {
	Success bool           `json:"success"`
	Data    any            `json:"data"`
	Error   *ErrorResponse `json:"error"`
}

func WriteOK(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{
		Success: true,
		Data:    data,
		Error:   nil,
	})
}

func WriteErr(c *gin.Context, status int, code string, message string) {
	c.JSON(status, Envelope{
		Success: false,
		Data:    nil,
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
		},
	})
}
