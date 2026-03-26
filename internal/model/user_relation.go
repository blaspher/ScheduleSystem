package model

import "time"

// UserRelation stores minimal permission for shared calendar viewing.
type UserRelation struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"not null;uniqueIndex:idx_user_target" json:"user_id"`
	TargetUserID    uint      `gorm:"not null;uniqueIndex:idx_user_target" json:"target_user_id"`
	CanViewCalendar bool      `gorm:"not null;default:false" json:"can_view_calendar"`
	CreatedAt       time.Time `json:"created_at"`
}

func (UserRelation) TableName() string {
	return "user_relations"
}
