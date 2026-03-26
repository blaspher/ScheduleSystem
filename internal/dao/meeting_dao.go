package dao

import (
	"context"
	"time"

	"schedule-system/internal/model"

	"gorm.io/gorm"
)

type MeetingInvitationRecord struct {
	MeetingID uint
	Role      string
	Status    string
	Event     model.Event
}

type MeetingDAO interface {
	CreateMeetingWithAttendees(ctx context.Context, event *model.Event, attendees []model.MeetingAttendee) error
	ListPendingInvitations(ctx context.Context, userID uint) ([]MeetingInvitationRecord, error)
	GetMeetingAttendee(ctx context.Context, meetingID, userID uint) (*model.MeetingAttendee, error)
	UpdateMeetingAttendeeStatus(ctx context.Context, meetingID, userID uint, status string) error
	GetMeetingEventByID(ctx context.Context, meetingID uint) (*model.Event, error)
	HasUserOwnedEventConflict(ctx context.Context, userID uint, startTime, endTime time.Time, excludeEventID *uint) (bool, error)
	HasAcceptedMeetingConflict(ctx context.Context, userID uint, startTime, endTime time.Time, excludeMeetingID uint) (bool, error)
}

type meetingDAO struct {
	db *gorm.DB
}

func NewMeetingDAO(db *gorm.DB) MeetingDAO {
	return &meetingDAO{db: db}
}

func (d *meetingDAO) CreateMeetingWithAttendees(ctx context.Context, event *model.Event, attendees []model.MeetingAttendee) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(event).Error; err != nil {
			return err
		}

		for i := range attendees {
			attendees[i].MeetingID = event.ID
		}
		if len(attendees) > 0 {
			if err := tx.Create(&attendees).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *meetingDAO) ListPendingInvitations(ctx context.Context, userID uint) ([]MeetingInvitationRecord, error) {
	type invitationRow struct {
		MeetingID   uint
		Role        string
		Status      string
		ID          uint
		OwnerID     uint
		Title       string
		Description string
		EventType   string
		Visibility  string
		StartTime   time.Time
		EndTime     time.Time
		Location    string
		EventStatus string
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}

	var rows []invitationRow
	if err := d.db.WithContext(ctx).
		Table("meeting_attendees AS ma").
		Select(`
			ma.meeting_id, ma.role, ma.status,
			e.id, e.owner_id, e.title, e.description, e.event_type, e.visibility,
			e.start_time, e.end_time, e.location, e.status AS event_status, e.created_at, e.updated_at
		`).
		Joins("JOIN events e ON e.id = ma.meeting_id").
		Where("ma.user_id = ?", userID).
		Where("ma.role = ?", model.MeetingRoleAttendee).
		Where("ma.status = ?", model.MeetingInviteStatusPending).
		Where("e.status <> ?", model.EventStatusCancelled).
		Where("e.event_type = ?", "meeting").
		Order("e.start_time ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]MeetingInvitationRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, MeetingInvitationRecord{
			MeetingID: row.MeetingID,
			Role:      row.Role,
			Status:    row.Status,
			Event: model.Event{
				ID:          row.ID,
				OwnerID:     row.OwnerID,
				Title:       row.Title,
				Description: row.Description,
				EventType:   row.EventType,
				Visibility:  row.Visibility,
				StartTime:   row.StartTime,
				EndTime:     row.EndTime,
				Location:    row.Location,
				Status:      row.EventStatus,
				CreatedAt:   row.CreatedAt,
				UpdatedAt:   row.UpdatedAt,
			},
		})
	}
	return result, nil
}

func (d *meetingDAO) GetMeetingAttendee(ctx context.Context, meetingID, userID uint) (*model.MeetingAttendee, error) {
	var attendee model.MeetingAttendee
	if err := d.db.WithContext(ctx).
		Where("meeting_id = ? AND user_id = ?", meetingID, userID).
		First(&attendee).Error; err != nil {
		return nil, err
	}
	return &attendee, nil
}

func (d *meetingDAO) UpdateMeetingAttendeeStatus(ctx context.Context, meetingID, userID uint, status string) error {
	result := d.db.WithContext(ctx).
		Model(&model.MeetingAttendee{}).
		Where("meeting_id = ? AND user_id = ?", meetingID, userID).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *meetingDAO) GetMeetingEventByID(ctx context.Context, meetingID uint) (*model.Event, error) {
	var event model.Event
	if err := d.db.WithContext(ctx).
		Where("id = ?", meetingID).
		Where("event_type = ?", "meeting").
		Where("status <> ?", model.EventStatusCancelled).
		First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (d *meetingDAO) HasUserOwnedEventConflict(ctx context.Context, userID uint, startTime, endTime time.Time, excludeEventID *uint) (bool, error) {
	query := d.db.WithContext(ctx).
		Model(&model.Event{}).
		Where("owner_id = ?", userID).
		Where("status <> ?", model.EventStatusCancelled).
		Where("start_time < ? AND end_time > ?", endTime, startTime)

	if excludeEventID != nil {
		query = query.Where("id <> ?", *excludeEventID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *meetingDAO) HasAcceptedMeetingConflict(ctx context.Context, userID uint, startTime, endTime time.Time, excludeMeetingID uint) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Table("meeting_attendees AS ma").
		Joins("JOIN events e ON e.id = ma.meeting_id").
		Where("ma.user_id = ?", userID).
		Where("ma.role = ?", model.MeetingRoleAttendee).
		Where("ma.status = ?", model.MeetingInviteStatusAccepted).
		Where("e.event_type = ?", "meeting").
		Where("e.status <> ?", model.EventStatusCancelled).
		Where("e.id <> ?", excludeMeetingID).
		Where("e.start_time < ? AND e.end_time > ?", endTime, startTime).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
