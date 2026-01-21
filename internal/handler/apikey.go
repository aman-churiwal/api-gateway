package handler

import (
	"net/http"

	"github.com/aman-churiwal/api-gateway/internal/service"
	"github.com/gin-gonic/gin"
)

type APIKeyHandler struct {
	service *service.APIKeyService
}

func NewAPIKeyHandler(service service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{service: &service}
}

func (h *APIKeyHandler) Create(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		CreatedBy string `json:"created_by"`
		Tier      string `json:"tier" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	key, err := h.service.Create(ctx, req.Name, req.CreatedBy, req.Tier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":     key,
		"message": "Save this key - it won't be shown again",
	})
}

func (h *APIKeyHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	keys, err := h.service.List(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, keys)
}

func (h *APIKeyHandler) Get(c *gin.Context) {
	id := c.Param("id")

	ctx := c.Request.Context()
	apiKey, err := h.service.Get(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if apiKey == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

func (h *APIKeyHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Tier     *string `json:"tier"`
		IsActive *bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Tier != nil {
		updates["tier"] = *req.Tier
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	ctx := c.Request.Context()
	if err := h.service.Update(ctx, id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key updated successfully"})
}

func (h *APIKeyHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	ctx := c.Request.Context()
	if err := h.service.Delete(ctx, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}
