package dao

import (
	"context"
	"errors"

	"schedule-system/internal/model"

	"gorm.io/gorm"
)

type UserRelationDAO interface {
	SetCalendarPermission(ctx context.Context, userID, targetUserID uint, canViewCalendar bool) (*model.UserRelation, error)
	CanViewCalendar(ctx context.Context, userID, targetUserID uint) (bool, error)
}

type userRelationDAO struct {
	db *gorm.DB
}

func NewUserRelationDAO(db *gorm.DB) UserRelationDAO {
	return &userRelationDAO{db: db}
}

func (d *userRelationDAO) SetCalendarPermission(ctx context.Context, userID, targetUserID uint, canViewCalendar bool) (*model.UserRelation, error) {
	var relation model.UserRelation
	err := d.db.WithContext(ctx).
		Where("user_id = ? AND target_user_id = ?", userID, targetUserID).
		First(&relation).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		relation = model.UserRelation{
			UserID:          userID,
			TargetUserID:    targetUserID,
			CanViewCalendar: canViewCalendar,
		}
		if err := d.db.WithContext(ctx).Create(&relation).Error; err != nil {
			return nil, err
		}
		return &relation, nil
	}

	relation.CanViewCalendar = canViewCalendar
	if err := d.db.WithContext(ctx).Model(&relation).
		Update("can_view_calendar", canViewCalendar).Error; err != nil {
		return nil, err
	}
	return &relation, nil
}

func (d *userRelationDAO) CanViewCalendar(ctx context.Context, userID, targetUserID uint) (bool, error) {
	var count int64
	if err := d.db.WithContext(ctx).
		Model(&model.UserRelation{}).
		Where("user_id = ? AND target_user_id = ? AND can_view_calendar = ?", userID, targetUserID, true).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
