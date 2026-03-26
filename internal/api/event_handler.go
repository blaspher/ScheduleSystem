package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"schedule-system/internal/model"
	"schedule-system/internal/service"

	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	eventService *service.EventService
}

type eventUpsertRequest struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	EventType   string    `json:"event_type"`
	Visibility  string    `json:"visibility" binding:"required"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
	Location    string    `json:"location"`
}

func NewEventHandler(eventService *service.EventService) *EventHandler {
	return &EventHandler{eventService: eventService}
}

func (h *EventHandler) Create(c *gin.Context) {
	ownerID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req eventUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid request payload")
		return
	}

	event, err := h.eventService.CreateEvent(c.Request.Context(), service.EventCreateInput{
		OwnerID:     ownerID,
		Title:       req.Title,
		Description: req.Description,
		EventType:   req.EventType,
		Visibility:  req.Visibility,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Location:    req.Location,
	})
	if err != nil {
		handleEventServiceError(c, err)
		return
	}

	successResponse(c, http.StatusCreated, event)
}

func (h *EventHandler) Update(c *gin.Context) {
	ownerID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := parseUintPathParam(c, "id")
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid event id")
		return
	}

	var req eventUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid request payload")
		return
	}

	event, err := h.eventService.UpdateEvent(c.Request.Context(), service.EventUpdateInput{
		ID:          eventID,
		OwnerID:     ownerID,
		Title:       req.Title,
		Description: req.Description,
		EventType:   req.EventType,
		Visibility:  req.Visibility,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Location:    req.Location,
	})
	if err != nil {
		handleEventServiceError(c, err)
		return
	}

	successResponse(c, http.StatusOK, event)
}

func (h *EventHandler) Delete(c *gin.Context) {
	ownerID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := parseUintPathParam(c, "id")
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid event id")
		return
	}

	if err := h.eventService.DeleteEvent(c.Request.Context(), ownerID, eventID); err != nil {
		handleEventServiceError(c, err)
		return
	}

	successResponse(c, http.StatusOK, gin.H{
		"id":     eventID,
		"status": model.EventStatusCancelled,
	})
}

func (h *EventHandler) GetByID(c *gin.Context) {
	ownerID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	eventID, err := parseUintPathParam(c, "id")
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid event id")
		return
	}

	event, err := h.eventService.GetEvent(c.Request.Context(), ownerID, eventID)
	if err != nil {
		handleEventServiceError(c, err)
		return
	}

	successResponse(c, http.StatusOK, event)
}

func (h *EventHandler) List(c *gin.Context) {
	ownerID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	input := service.EventListInput{
		OwnerID:          ownerID,
		IncludeCancelled: strings.EqualFold(c.Query("include_cancelled"), "true"),
	}

	if rawFrom := c.Query("start_time_from"); rawFrom != "" {
		v, err := time.Parse(time.RFC3339, rawFrom)
		if err != nil {
			errorResponse(c, http.StatusBadRequest, "invalid start_time_from, must be RFC3339")
			return
		}
		input.StartTimeFrom = &v
	}
	if rawTo := c.Query("start_time_to"); rawTo != "" {
		v, err := time.Parse(time.RFC3339, rawTo)
		if err != nil {
			errorResponse(c, http.StatusBadRequest, "invalid start_time_to, must be RFC3339")
			return
		}
		input.StartTimeTo = &v
	}

	input.View = c.Query("view")
	if rawDate := c.Query("date"); rawDate != "" {
		v, err := time.ParseInLocation("2006-01-02", rawDate, time.Local)
		if err != nil {
			errorResponse(c, http.StatusBadRequest, "invalid date, must be YYYY-MM-DD")
			return
		}
		input.Date = &v
	}

	events, err := h.eventService.ListEvents(c.Request.Context(), input)
	if err != nil {
		handleEventServiceError(c, err)
		return
	}

	successResponse(c, http.StatusOK, events)
}

func ownerIDFromContext(c *gin.Context) (uint, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		return 0, false
	}
	return userID, true
}

func parseUintPathParam(c *gin.Context, key string) (uint, error) {
	idStr := c.Param(key)
	idVal, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(idVal), nil
}

func handleEventServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrEventNotFound):
		errorResponse(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidTimeRange),
		errors.Is(err, service.ErrInvalidVisibility),
		errors.Is(err, service.ErrInvalidView),
		errors.Is(err, service.ErrDateRequired),
		errors.Is(err, service.ErrInvalidFilterTimeRange):
		errorResponse(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrEventTimeConflict):
		errorResponse(c, http.StatusConflict, err.Error())
	default:
		errorResponse(c, http.StatusInternalServerError, "internal server error")
	}
}
