package api

import (
	"errors"
	"net/http"

	"schedule-system/internal/service"

	"github.com/gin-gonic/gin"
)

type RelationHandler struct {
	relationService *service.RelationService
}

type setRelationPermissionRequest struct {
	TargetUserID    uint  `json:"target_user_id" binding:"required"`
	CanViewCalendar *bool `json:"can_view_calendar" binding:"required"`
}

func NewRelationHandler(relationService *service.RelationService) *RelationHandler {
	return &RelationHandler{relationService: relationService}
}

func (h *RelationHandler) SetCalendarPermission(c *gin.Context) {
	userID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setRelationPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid request payload")
		return
	}

	result, err := h.relationService.SetCalendarPermission(c.Request.Context(), service.SetRelationPermissionInput{
		UserID:          userID,
		TargetUserID:    req.TargetUserID,
		CanViewCalendar: *req.CanViewCalendar,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidTargetUser) {
			errorResponse(c, http.StatusBadRequest, err.Error())
			return
		}
		errorResponse(c, http.StatusInternalServerError, "internal server error")
		return
	}

	successResponse(c, http.StatusOK, result)
}
