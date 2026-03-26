package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"schedule-system/internal/dao"
	"schedule-system/internal/model"
	cachepkg "schedule-system/pkg/cache"

	"gorm.io/gorm"
)

var ErrInvalidAttendeeIDs = errors.New("attendee_ids must not be empty")
var ErrInvalidAttendeeID = errors.New("attendee_id must be greater than 0")
var ErrSelfInAttendeeIDs = errors.New("current user must not appear in attendee_ids")
var ErrMeetingNotFound = errors.New("meeting not found")
var ErrInvitationNotFound = errors.New("invitation not found")
var ErrInvalidInvitationState = errors.New("invalid invitation state")
var ErrMeetingTimeConflict = errors.New("meeting time conflict")

type MeetingService struct {
	meetingDAO dao.MeetingDAO
	cacheStore *cachepkg.Store
}

type CreateMeetingInput struct {
	OrganizerID uint
	Title       string
	Description string
	Visibility  string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
	AttendeeIDs []uint
}

type MeetingAttendeeResult struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

type CreateMeetingResult struct {
	MeetingID uint                  `json:"meeting_id"`
	Event     model.Event           `json:"event"`
	Attendees []MeetingAttendeeResult `json:"attendees"`
}

type MeetingInvitationItem struct {
	MeetingID uint       `json:"meeting_id"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	Event     model.Event `json:"event"`
}

type MeetingInvitationStateResult struct {
	MeetingID uint   `json:"meeting_id"`
	Status    string `json:"status"`
}

func NewMeetingService(meetingDAO dao.MeetingDAO, cacheStore *cachepkg.Store) *MeetingService {
	return &MeetingService{
		meetingDAO: meetingDAO,
		cacheStore: cacheStore,
	}
}

func (s *MeetingService) CreateMeeting(ctx context.Context, input CreateMeetingInput) (*CreateMeetingResult, error) {
	if err := validateEventTimeRange(input.StartTime, input.EndTime); err != nil {
		return nil, err
	}
	if !isMeetingVisibilityValid(input.Visibility) {
		return nil, ErrInvalidVisibility
	}

	attendeeIDs, err := normalizeAttendeeIDs(input.OrganizerID, input.AttendeeIDs)
	if err != nil {
		return nil, err
	}

	conflict, err := s.meetingDAO.HasUserOwnedEventConflict(ctx, input.OrganizerID, input.StartTime, input.EndTime, nil)
	if err != nil {
		return nil, err
	}
	if conflict {
		return nil, ErrMeetingTimeConflict
	}
	conflictWithAcceptedMeetings, err := s.meetingDAO.HasAcceptedMeetingConflict(ctx, input.OrganizerID, input.StartTime, input.EndTime, 0)
	if err != nil {
		return nil, err
	}
	if conflictWithAcceptedMeetings {
		return nil, ErrMeetingTimeConflict
	}

	event := &model.Event{
		OwnerID:     input.OrganizerID,
		Title:       strings.TrimSpace(input.Title),
		Description: input.Description,
		EventType:   "meeting",
		Visibility:  input.Visibility,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		Location:    input.Location,
		Status:      model.EventStatusActive,
	}

	attendees := make([]model.MeetingAttendee, 0, len(attendeeIDs)+1)
	attendees = append(attendees, model.MeetingAttendee{
		UserID: input.OrganizerID,
		Role:   model.MeetingRoleOrganizer,
		Status: model.MeetingInviteStatusAccepted,
	})
	for _, attendeeID := range attendeeIDs {
		attendees = append(attendees, model.MeetingAttendee{
			UserID: attendeeID,
			Role:   model.MeetingRoleAttendee,
			Status: model.MeetingInviteStatusPending,
		})
	}

	if err := s.meetingDAO.CreateMeetingWithAttendees(ctx, event, attendees); err != nil {
		return nil, err
	}
	s.invalidateOrganizerCaches(ctx, input.OrganizerID, event.ID)

	resultAttendees := make([]MeetingAttendeeResult, 0, len(attendees))
	for _, attendee := range attendees {
		resultAttendees = append(resultAttendees, MeetingAttendeeResult{
			UserID: attendee.UserID,
			Role:   attendee.Role,
			Status: attendee.Status,
		})
	}

	return &CreateMeetingResult{
		MeetingID: event.ID,
		Event:     *event,
		Attendees: resultAttendees,
	}, nil
}

func (s *MeetingService) ListPendingInvitations(ctx context.Context, userID uint) ([]MeetingInvitationItem, error) {
	rows, err := s.meetingDAO.ListPendingInvitations(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]MeetingInvitationItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, MeetingInvitationItem{
			MeetingID: row.MeetingID,
			Role:      row.Role,
			Status:    row.Status,
			Event:     row.Event,
		})
	}
	return result, nil
}

func (s *MeetingService) AcceptInvitation(ctx context.Context, userID, meetingID uint) (*MeetingInvitationStateResult, error) {
	attendee, meeting, err := s.ensurePendingAttendeeInvitation(ctx, userID, meetingID)
	if err != nil {
		return nil, err
	}

	conflictWithOwnedEvents, err := s.meetingDAO.HasUserOwnedEventConflict(ctx, userID, meeting.StartTime, meeting.EndTime, nil)
	if err != nil {
		return nil, err
	}
	if conflictWithOwnedEvents {
		return nil, ErrMeetingTimeConflict
	}

	conflictWithAcceptedMeetings, err := s.meetingDAO.HasAcceptedMeetingConflict(ctx, userID, meeting.StartTime, meeting.EndTime, meetingID)
	if err != nil {
		return nil, err
	}
	if conflictWithAcceptedMeetings {
		return nil, ErrMeetingTimeConflict
	}

	if err := s.meetingDAO.UpdateMeetingAttendeeStatus(ctx, attendee.MeetingID, attendee.UserID, model.MeetingInviteStatusAccepted); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	return &MeetingInvitationStateResult{
		MeetingID: meetingID,
		Status:    model.MeetingInviteStatusAccepted,
	}, nil
}

func (s *MeetingService) RejectInvitation(ctx context.Context, userID, meetingID uint) (*MeetingInvitationStateResult, error) {
	attendee, _, err := s.ensurePendingAttendeeInvitation(ctx, userID, meetingID)
	if err != nil {
		return nil, err
	}

	if err := s.meetingDAO.UpdateMeetingAttendeeStatus(ctx, attendee.MeetingID, attendee.UserID, model.MeetingInviteStatusRejected); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	return &MeetingInvitationStateResult{
		MeetingID: meetingID,
		Status:    model.MeetingInviteStatusRejected,
	}, nil
}

func (s *MeetingService) ensurePendingAttendeeInvitation(ctx context.Context, userID, meetingID uint) (*model.MeetingAttendee, *model.Event, error) {
	attendee, err := s.meetingDAO.GetMeetingAttendee(ctx, meetingID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvitationNotFound
		}
		return nil, nil, err
	}
	if attendee.Role != model.MeetingRoleAttendee {
		return nil, nil, ErrInvitationNotFound
	}
	if attendee.Status != model.MeetingInviteStatusPending {
		return nil, nil, ErrInvalidInvitationState
	}

	meeting, err := s.meetingDAO.GetMeetingEventByID(ctx, meetingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrMeetingNotFound
		}
		return nil, nil, err
	}
	return attendee, meeting, nil
}

func normalizeAttendeeIDs(organizerID uint, rawIDs []uint) ([]uint, error) {
	if len(rawIDs) == 0 {
		return nil, ErrInvalidAttendeeIDs
	}

	seen := make(map[uint]struct{}, len(rawIDs))
	result := make([]uint, 0, len(rawIDs))
	for _, attendeeID := range rawIDs {
		if attendeeID == 0 {
			return nil, ErrInvalidAttendeeID
		}
		if attendeeID == organizerID {
			return nil, ErrSelfInAttendeeIDs
		}
		if _, exists := seen[attendeeID]; exists {
			continue
		}
		seen[attendeeID] = struct{}{}
		result = append(result, attendeeID)
	}

	if len(result) == 0 {
		return nil, ErrInvalidAttendeeIDs
	}
	return result, nil
}

func isMeetingVisibilityValid(visibility string) bool {
	switch visibility {
	case model.EventVisibilityPrivate, model.EventVisibilityBusyOnly, model.EventVisibilityPublic:
		return true
	default:
		return false
	}
}

func (s *MeetingService) invalidateOrganizerCaches(ctx context.Context, organizerID, meetingID uint) {
	if s.cacheStore == nil {
		return
	}

	_ = s.cacheStore.DeleteByPattern(ctx, cachepkg.EventListPattern(organizerID))
	_ = s.cacheStore.DeleteByPattern(ctx, cachepkg.CalendarOwnerPattern(organizerID))
	_ = s.cacheStore.Delete(ctx, cachepkg.EventItemKey(organizerID, meetingID))
}
