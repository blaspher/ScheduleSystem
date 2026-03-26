package service

import (
	"context"
	"errors"
	"log"
	"time"

	"schedule-system/internal/dao"
	"schedule-system/internal/model"
	cachepkg "schedule-system/pkg/cache"
)

var ErrCalendarForbidden = errors.New("forbidden")
var ErrCalendarInvalidView = errors.New("invalid view, must be day or week")

type CalendarService struct {
	userRelationDAO dao.UserRelationDAO
	eventDAO        dao.EventDAO
	cacheStore      *cachepkg.Store
}

type GetSharedCalendarInput struct {
	ViewerID uint
	OwnerID  uint
	View     string
	Date     time.Time
}

type SharedCalendarResult struct {
	OwnerID uint          `json:"owner_id"`
	View    string        `json:"view"`
	Date    string        `json:"date"`
	Events  []model.Event `json:"events"`
}

func NewCalendarService(userRelationDAO dao.UserRelationDAO, eventDAO dao.EventDAO, cacheStore *cachepkg.Store) *CalendarService {
	return &CalendarService{
		userRelationDAO: userRelationDAO,
		eventDAO:        eventDAO,
		cacheStore:      cacheStore,
	}
}

func (s *CalendarService) GetSharedCalendar(ctx context.Context, input GetSharedCalendarInput) (*SharedCalendarResult, error) {
	if input.View != "day" && input.View != "week" {
		return nil, ErrCalendarInvalidView
	}

	if input.ViewerID != input.OwnerID {
		canView, err := s.userRelationDAO.CanViewCalendar(ctx, input.ViewerID, input.OwnerID)
		if err != nil {
			return nil, err
		}
		if !canView {
			return nil, ErrCalendarForbidden
		}
	}

	cacheKey := cachepkg.CalendarKey(input.ViewerID, input.OwnerID, input.View, input.Date)
	if s.cacheStore != nil {
		var cached SharedCalendarResult
		cacheHit, err := s.cacheStore.GetJSON(ctx, cacheKey, &cached)
		if err != nil {
			log.Printf("cache read failed for calendar key=%s: %v", cacheKey, err)
			if errors.Is(err, cachepkg.ErrCacheDecode) {
				if delErr := s.cacheStore.Delete(ctx, cacheKey); delErr != nil {
					log.Printf("cache delete failed for corrupted calendar key=%s: %v", cacheKey, delErr)
				}
			}
		} else if cacheHit {
			return &cached, nil
		}
	}

	start, end := calendarRange(input.View, input.Date)
	events, err := s.eventDAO.ListByOwnerID(ctx, input.OwnerID, dao.EventListFilter{
		StartTimeFrom:    &start,
		StartTimeTo:      &end,
		IncludeCancelled: false,
	})
	if err != nil {
		return nil, err
	}

	filtered := filterEventsByVisibility(events)
	result := &SharedCalendarResult{
		OwnerID: input.OwnerID,
		View:    input.View,
		Date:    input.Date.Format("2006-01-02"),
		Events:  filtered,
	}
	if s.cacheStore != nil {
		if err := s.cacheStore.SetJSON(ctx, cacheKey, result); err != nil {
			log.Printf("cache write failed for calendar key=%s: %v", cacheKey, err)
		}
	}
	return result, nil
}

func calendarRange(view string, date time.Time) (time.Time, time.Time) {
	dayStart := atDayStart(date)
	if view == "day" {
		return dayStart, dayStart.AddDate(0, 0, 1)
	}

	weekStart := startOfWeek(dayStart)
	return weekStart, weekStart.AddDate(0, 0, 7)
}

func filterEventsByVisibility(events []model.Event) []model.Event {
	result := make([]model.Event, 0, len(events))
	for _, event := range events {
		switch event.Visibility {
		case model.EventVisibilityPrivate:
			continue
		case model.EventVisibilityBusyOnly:
			event.Title = "Busy"
			event.Description = ""
			event.EventType = ""
			event.Location = ""
			result = append(result, event)
		case model.EventVisibilityPublic:
			result = append(result, event)
		}
	}
	return result
}
