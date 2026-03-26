package model

import "time"

const (
	MeetingRoleOrganizer = "organizer"
	MeetingRoleAttendee  = "attendee"
)

const (
	MeetingInviteStatusPending  = "pending"
	MeetingInviteStatusAccepted = "accepted"
	MeetingInviteStatusRejected = "rejected"
)

// MeetingAttendee stores participant state for meeting events.
type MeetingAttendee struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MeetingID uint      `gorm:"not null;index;uniqueIndex:idx_meeting_user" json:"meeting_id"`
	UserID    uint      `gorm:"not null;index;uniqueIndex:idx_meeting_user" json:"user_id"`
	Role      string    `gorm:"size:32;not null;index" json:"role"`
	Status    string    `gorm:"size:32;not null;index" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (MeetingAttendee) TableName() string {
	return "meeting_attendees"
}
