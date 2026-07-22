package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	rest "github.com/yuudev14/ytsoar/internal/adapters/http/rests"
	"github.com/yuudev14/ytsoar/internal/application/auth"
)

func (h *AdminHandler) ListAuthProviders(c *gin.Context) {
	response := rest.Response{C: c}

	providers, err := h.authService.ListProvidersAdmin(c.Request.Context())
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(providers)
}

func (h *AdminHandler) CreateAuthProvider(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, ok := h.actor(c)
	if !ok {
		return
	}

	var body auth.ProviderInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	provider, err := h.authService.CreateProvider(c.Request.Context(), actorID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	c.IndentedJSON(http.StatusCreated, provider)
}

func (h *AdminHandler) UpdateAuthProvider(c *gin.Context) {
	response := rest.Response{C: c}

	actorID, providerID, ok := h.actorAndTarget(c, "provider_id")
	if !ok {
		return
	}

	var body auth.UpdateProviderInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.ResponseError(http.StatusBadRequest, err.Error())
		return
	}

	provider, err := h.authService.UpdateProvider(c.Request.Context(), actorID, providerID, body)
	if err != nil {
		response.Fail(h.logger, err)
		return
	}
	response.ResponseSuccess(provider)
}
