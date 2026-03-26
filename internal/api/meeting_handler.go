package api

import (
	"errors"
	"net/http"
	"time"

	"schedule-system/internal/service"

	"github.com/gin-gonic/gin"
)

type MeetingHandler struct {
	meetingService *service.MeetingService
}

type createMeetingRequest struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	Visibility  string    `json:"visibility" binding:"required"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
	Location    string    `json:"location"`
	AttendeeIDs []uint    `json:"attendee_ids" binding:"required"`
}

func NewMeetingHandler(meetingService *service.MeetingService) *MeetingHandler {
	return &MeetingHandler{meetingService: meetingService}
}

func (h *MeetingHandler) Create(c *gin.Context) {
	userID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createMeetingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid request payload")
		return
	}

	result, err := h.meetingService.CreateMeeting(c.Request.Context(), service.CreateMeetingInput{
		OrganizerID: userID,
		Title:       req.Title,
		Description: req.Description,
		Visibility:  req.Visibility,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Location:    req.Location,
		AttendeeIDs: req.AttendeeIDs,
	})
	if err != nil {
		handleMeetingServiceError(c, err)
		return
	}

	successResponse(c, http.StatusCreated, result)
}

func (h *MeetingHandler) ListInvitations(c *gin.Context) {
	userID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	result, err := h.meetingService.ListPendingInvitations(c.Request.Context(), userID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "internal server error")
		return
	}

	successResponse(c, http.StatusOK, result)
}

func (h *MeetingHandler) Accept(c *gin.Context) {
	userID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	meetingID, err := parseUintPathParam(c, "id")
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid meeting id")
		return
	}

	result, err := h.meetingService.AcceptInvitation(c.Request.Context(), userID, meetingID)
	if err != nil {
		handleMeetingServiceError(c, err)
		return
	}

	successResponse(c, http.StatusOK, result)
}

func (h *MeetingHandler) Reject(c *gin.Context) {
	userID, ok := ownerIDFromContext(c)
	if !ok {
		errorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	meetingID, err := parseUintPathParam(c, "id")
	if err != nil {
		errorResponse(c, http.StatusBadRequest, "invalid meeting id")
		return
	}

	result, err := h.meetingService.RejectInvitation(c.Request.Context(), userID, meetingID)
	if err != nil {
		handleMeetingServiceError(c, err)
		return
	}

	successResponse(c, http.StatusOK, result)
}

func handleMeetingServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidAttendeeIDs),
		errors.Is(err, service.ErrInvalidAttendeeID),
		errors.Is(err, service.ErrSelfInAttendeeIDs),
		errors.Is(err, service.ErrInvalidTimeRange),
		errors.Is(err, service.ErrInvalidVisibility),
		errors.Is(err, service.ErrInvalidInvitationState):
		errorResponse(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvitationNotFound),
		errors.Is(err, service.ErrMeetingNotFound):
		errorResponse(c, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrMeetingTimeConflict):
		errorResponse(c, http.StatusConflict, err.Error())
	default:
		errorResponse(c, http.StatusInternalServerError, "internal server error")
	}
}
