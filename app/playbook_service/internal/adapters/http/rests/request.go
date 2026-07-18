package rest

import (
	"net/http"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
)

// function for validating the request body
func ValidateData[T any](valid validation.Validation, data T) (bool, int, interface{}) {

	check, validErr := valid.Valid(data)

	if validErr != nil {
		return false, http.StatusUnprocessableEntity, valid.Errors
	}

	if !check {
		return false, http.StatusInternalServerError, valid.Errors
	}

	return true, http.StatusOK, nil

}

// binding and validating request body
func BindFormAndValidate[T any](c *gin.Context, form *T) (bool, int, interface{}) {
	err := c.ShouldBindJSON(&form)

	if err != nil {
		return false, http.StatusBadRequest, err.Error()
	}

	valid := validation.Validation{}

	return ValidateData(valid, form)

}

// binding and validating request body
func BindQueryAndValidate[T any](c *gin.Context, form *T) (bool, int, interface{}) {
	err := c.ShouldBindQuery(&form)

	if err != nil {
		return false, http.StatusBadRequest, err.Error()
	}

	valid := validation.Validation{}

	return ValidateData(valid, form)

}
