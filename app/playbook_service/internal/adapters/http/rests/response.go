package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	C *gin.Context
}

// function for sending a response error
func (r *Response) ResponseError(code int, data interface{}) {
	r.C.IndentedJSON(code, gin.H{
		"error": data,
	})
}

// function for sending a response
func (r *Response) Response(code int, data interface{}) {
	r.C.IndentedJSON(code, data)
}

// function for sending a response success
func (r *Response) ResponseSuccess(data interface{}) {
	r.C.IndentedJSON(http.StatusOK, data)
}
