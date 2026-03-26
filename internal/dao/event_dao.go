package dao

import (
	"context"
	"time"

	"schedule-system/internal/model"

	"gorm.io/gorm"
)

type EventListFilter struct {
	StartTimeFrom     *time.Time
	StartTimeTo       *time.Time
	IncludeCancelled  bool
}

type EventDAO interface {
	Create(ctx context.Context, event *model.Event) error
	Update(ctx context.Context, event *model.Event) error
	GetByIDAndOwnerID(ctx context.Context, id, ownerID uint) (*model.Event, error)
	ListByOwnerID(ctx context.Context, ownerID uint, filter EventListFilter) ([]model.Event, error)
	SoftDelete(ctx context.Context, id, ownerID uint) error
	HasConflict(ctx context.Context, ownerID uint, startTime, endTime time.Time, excludeID *uint) (bool, error)
}

type eventDAO struct {
	db *gorm.DB
}

func NewEventDAO(db *gorm.DB) EventDAO {
	return &eventDAO{db: db}
}

func (d *eventDAO) Create(ctx context.Context, event *model.Event) error {
	return d.db.WithContext(ctx).Create(event).Error
}

func (d *eventDAO) Update(ctx context.Context, event *model.Event) error {
	return d.db.WithContext(ctx).Save(event).Error
}

func (d *eventDAO) GetByIDAndOwnerID(ctx context.Context, id, ownerID uint) (*model.Event, error) {
	var event model.Event
	if err := d.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, ownerID).
		First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (d *eventDAO) ListByOwnerID(ctx context.Context, ownerID uint, filter EventListFilter) ([]model.Event, error) {
	query := d.db.WithContext(ctx).Where("owner_id = ?", ownerID)
	if !filter.IncludeCancelled {
		query = query.Where("status <> ?", model.EventStatusCancelled)
	}
	if filter.StartTimeFrom != nil {
		query = query.Where("start_time >= ?", *filter.StartTimeFrom)
	}
	if filter.StartTimeTo != nil {
		query = query.Where("start_time < ?", *filter.StartTimeTo)
	}

	var events []model.Event
	if err := query.Order("start_time ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (d *eventDAO) SoftDelete(ctx context.Context, id, ownerID uint) error {
	result := d.db.WithContext(ctx).
		Model(&model.Event{}).
		Where("id = ? AND owner_id = ?", id, ownerID).
		Update("status", model.EventStatusCancelled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *eventDAO) HasConflict(ctx context.Context, ownerID uint, startTime, endTime time.Time, excludeID *uint) (bool, error) {
	query := d.db.WithContext(ctx).
		Model(&model.Event{}).
		Where("owner_id = ?", ownerID).
		Where("status <> ?", model.EventStatusCancelled).
		Where("start_time < ? AND end_time > ?", endTime, startTime)

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
