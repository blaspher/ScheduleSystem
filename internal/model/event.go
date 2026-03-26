package model

import "time"

const (
	EventVisibilityPrivate  = "private"
	EventVisibilityBusyOnly = "busy_only"
	EventVisibilityPublic   = "public"
)

const (
	EventStatusActive    = "active"
	EventStatusCancelled = "cancelled"
)

// Event represents a calendar event owned by a user.
type Event struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OwnerID     uint      `gorm:"index;not null" json:"owner_id"`
	Title       string    `gorm:"size:255;not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	EventType   string    `gorm:"size:64;index" json:"event_type"`
	Visibility  string    `gorm:"size:32;index;not null;default:private" json:"visibility"`
	StartTime   time.Time `gorm:"index;not null" json:"start_time"`
	EndTime     time.Time `gorm:"index;not null" json:"end_time"`
	Location    string    `gorm:"size:255" json:"location"`
	Status      string    `gorm:"size:32;index;not null;default:active" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
