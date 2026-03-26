package api

import (
	"errors"
	"net/http"
	"time"

	"schedule-system/internal/service"

	"github.com/gin-gonic/gin"
)

type CalendarHandler struct {
	calendarService *service.CalendarService
}

func NewCalendarHandler(calendarService *service.CalendarService) *CalendarHandler {
	return &CalendarHandler{calendarService: calendarService}
}

func (h *CalendarHandler) GetUserCalendar(c *gin.Context) {
	viewerID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	ownerID, err := parseUintPathParam(c, "id")
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid user id")
		return
	}

	view := c.Query("view")
	if view == "" {
		errorResponse(c, http.StatusBadRequest, "view is required")
		return
	}

	rawDate := c.Query("date")
	if rawDate == "" {
		errorResponse(c, http.StatusBadRequest, "date is required")
		return
	}
	date, err := time.ParseInLocation("2006-01-02", rawDate, time.Local)
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid date, must be YYYY-MM-DD")
		return
	}

	result, err := h.calendarService.GetSharedCalendar(c.Request.Context(), service.GetSharedCalendarInput{
		ViewerID: viewerID,
		OwnerID:  ownerID,
		View:     view,
		Date:     date,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCalendarInvalidView):
			errorResponse(c, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrCalendarForbidden):
			errorResponse(c, http.StatusForbidden, err.Error())
		default:
			errorResponse(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	successResponse(c, http.StatusOK, result)
}
