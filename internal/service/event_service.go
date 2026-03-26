package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"schedule-system/internal/dao"
	"schedule-system/internal/model"
	cachepkg "schedule-system/pkg/cache"

	"gorm.io/gorm"
)

var ErrEventNotFound = errors.New("event not found")
var ErrInvalidTimeRange = errors.New("start_time must be before end_time")
var ErrEventTimeConflict = errors.New("event time conflict")
var ErrInvalidVisibility = errors.New("invalid visibility")
var ErrInvalidView = errors.New("invalid view, must be day or week")
var ErrDateRequired = errors.New("date is required when view is set")
var ErrInvalidFilterTimeRange = errors.New("start_time_from must be before start_time_to")

type EventService struct {
	eventDAO   dao.EventDAO
	cacheStore *cachepkg.Store
}

type EventCreateInput struct {
	OwnerID     uint
	Title       string
	Description string
	EventType   string
	Visibility  string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
}

type EventUpdateInput struct {
	ID          uint
	OwnerID     uint
	Title       string
	Description string
	EventType   string
	Visibility  string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
}

type EventListInput struct {
	OwnerID           uint
	StartTimeFrom     *time.Time
	StartTimeTo       *time.Time
	View              string
	Date              *time.Time
	IncludeCancelled  bool
}

func NewEventService(eventDAO dao.EventDAO, cacheStore *cachepkg.Store) *EventService {
	return &EventService{
		eventDAO:   eventDAO,
		cacheStore: cacheStore,
	}
}

func (s *EventService) CreateEvent(ctx context.Context, input EventCreateInput) (*model.Event, error) {
	if err := validateEventTimeRange(input.StartTime, input.EndTime); err != nil {
		return nil, err
	}
	if !isValidVisibility(input.Visibility) {
		return nil, ErrInvalidVisibility
	}

	conflict, err := s.eventDAO.HasConflict(ctx, input.OwnerID, input.StartTime, input.EndTime, nil)
	if err != nil {
		return nil, err
	}
	if conflict {
		return nil, ErrEventTimeConflict
	}

	event := &model.Event{
		OwnerID:     input.OwnerID,
		Title:       strings.TrimSpace(input.Title),
		Description: input.Description,
		EventType:   input.EventType,
		Visibility:  input.Visibility,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		Location:    input.Location,
		Status:      model.EventStatusActive,
	}
	if err := s.eventDAO.Create(ctx, event); err != nil {
		return nil, err
	}
	s.invalidateOwnerCaches(ctx, input.OwnerID, &event.ID)
	return event, nil
}

func (s *EventService) UpdateEvent(ctx context.Context, input EventUpdateInput) (*model.Event, error) {
	if err := validateEventTimeRange(input.StartTime, input.EndTime); err != nil {
		return nil, err
	}
	if !isValidVisibility(input.Visibility) {
		return nil, ErrInvalidVisibility
	}

	event, err := s.eventDAO.GetByIDAndOwnerID(ctx, input.ID, input.OwnerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	if event.Status == model.EventStatusCancelled {
		return nil, ErrEventNotFound
	}

	conflict, err := s.eventDAO.HasConflict(ctx, input.OwnerID, input.StartTime, input.EndTime, &input.ID)
	if err != nil {
		return nil, err
	}
	if conflict {
		return nil, ErrEventTimeConflict
	}

	event.Title = strings.TrimSpace(input.Title)
	event.Description = input.Description
	event.EventType = input.EventType
	event.Visibility = input.Visibility
	event.StartTime = input.StartTime
	event.EndTime = input.EndTime
	event.Location = input.Location

	if err := s.eventDAO.Update(ctx, event); err != nil {
		return nil, err
	}
	s.invalidateOwnerCaches(ctx, input.OwnerID, &event.ID)
	return event, nil
}

func (s *EventService) DeleteEvent(ctx context.Context, ownerID, eventID uint) error {
	if err := s.eventDAO.SoftDelete(ctx, eventID, ownerID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEventNotFound
		}
		return err
	}
	s.invalidateOwnerCaches(ctx, ownerID, &eventID)
	return nil
}

func (s *EventService) GetEvent(ctx context.Context, ownerID, eventID uint) (*model.Event, error) {
	if s.cacheStore != nil {
		var cached model.Event
		cacheHit, err := s.cacheStore.GetJSON(ctx, cachepkg.EventItemKey(ownerID, eventID), &cached)
		if err != nil {
			log.Printf("cache read failed for event item key=%s: %v", cachepkg.EventItemKey(ownerID, eventID), err)
			if errors.Is(err, cachepkg.ErrCacheDecode) {
				if delErr := s.cacheStore.Delete(ctx, cachepkg.EventItemKey(ownerID, eventID)); delErr != nil {
					log.Printf("cache delete failed for corrupted event item key=%s: %v", cachepkg.EventItemKey(ownerID, eventID), delErr)
				}
			}
		} else if cacheHit {
			return &cached, nil
		}
	}

	event, err := s.eventDAO.GetByIDAndOwnerID(ctx, eventID, ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	if event.Status == model.EventStatusCancelled {
		return nil, ErrEventNotFound
	}

	if s.cacheStore != nil {
		if err := s.cacheStore.SetJSON(ctx, cachepkg.EventItemKey(ownerID, eventID), event); err != nil {
			log.Printf("cache write failed for event item key=%s: %v", cachepkg.EventItemKey(ownerID, eventID), err)
		}
	}
	return event, nil
}

func (s *EventService) ListEvents(ctx context.Context, input EventListInput) ([]model.Event, error) {
	cacheKey := cachepkg.EventListKey(
		input.OwnerID,
		input.StartTimeFrom,
		input.StartTimeTo,
		input.View,
		input.Date,
		input.IncludeCancelled,
	)
	if s.cacheStore != nil {
		var cached []model.Event
		cacheHit, err := s.cacheStore.GetJSON(ctx, cacheKey, &cached)
		if err != nil {
			log.Printf("cache read failed for event list key=%s: %v", cacheKey, err)
			if errors.Is(err, cachepkg.ErrCacheDecode) {
				if delErr := s.cacheStore.Delete(ctx, cacheKey); delErr != nil {
					log.Printf("cache delete failed for corrupted event list key=%s: %v", cacheKey, delErr)
				}
			}
		} else if cacheHit {
			return cached, nil
		}
	}

	filter := dao.EventListFilter{
		IncludeCancelled: input.IncludeCancelled,
	}

	if input.StartTimeFrom != nil || input.StartTimeTo != nil {
		if input.StartTimeFrom != nil && input.StartTimeTo != nil && !input.StartTimeFrom.Before(*input.StartTimeTo) {
			return nil, ErrInvalidFilterTimeRange
		}
		filter.StartTimeFrom = input.StartTimeFrom
		filter.StartTimeTo = input.StartTimeTo
		events, err := s.eventDAO.ListByOwnerID(ctx, input.OwnerID, filter)
		if err != nil {
			return nil, err
		}
		if s.cacheStore != nil {
			if err := s.cacheStore.SetJSON(ctx, cacheKey, events); err != nil {
				log.Printf("cache write failed for event list key=%s: %v", cacheKey, err)
			}
		}
		return events, nil
	}

	if input.View != "" {
		if input.Date == nil {
			return nil, ErrDateRequired
		}

		dayStart := atDayStart(*input.Date)
		switch input.View {
		case "day":
			dayEnd := dayStart.AddDate(0, 0, 1)
			filter.StartTimeFrom = &dayStart
			filter.StartTimeTo = &dayEnd
		case "week":
			weekStart := startOfWeek(dayStart)
			weekEnd := weekStart.AddDate(0, 0, 7)
			filter.StartTimeFrom = &weekStart
			filter.StartTimeTo = &weekEnd
		default:
			return nil, ErrInvalidView
		}
	}

	events, err := s.eventDAO.ListByOwnerID(ctx, input.OwnerID, filter)
	if err != nil {
		return nil, err
	}
	if s.cacheStore != nil {
		if err := s.cacheStore.SetJSON(ctx, cacheKey, events); err != nil {
			log.Printf("cache write failed for event list key=%s: %v", cacheKey, err)
		}
	}
	return events, nil
}

func validateEventTimeRange(startTime, endTime time.Time) error {
	if !startTime.Before(endTime) {
		return ErrInvalidTimeRange
	}
	return nil
}

func isValidVisibility(v string) bool {
	switch v {
	case model.EventVisibilityPrivate, model.EventVisibilityBusyOnly, model.EventVisibilityPublic:
		return true
	default:
		return false
	}
}

func atDayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}

func (s *EventService) invalidateOwnerCaches(ctx context.Context, ownerID uint, eventID *uint) {
	if s.cacheStore == nil {
		return
	}

	_ = s.cacheStore.DeleteByPattern(ctx, cachepkg.EventListPattern(ownerID))
	_ = s.cacheStore.DeleteByPattern(ctx, cachepkg.CalendarOwnerPattern(ownerID))
	if eventID != nil {
		_ = s.cacheStore.Delete(ctx, cachepkg.EventItemKey(ownerID, *eventID))
	}
}
