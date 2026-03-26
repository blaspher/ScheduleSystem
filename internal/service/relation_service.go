package service

import (
	"context"
	"errors"

	"schedule-system/internal/dao"
	cachepkg "schedule-system/pkg/cache"
)

var ErrInvalidTargetUser = errors.New("invalid target_user_id")

type RelationService struct {
	userRelationDAO dao.UserRelationDAO
	cacheStore      *cachepkg.Store
}

type SetRelationPermissionInput struct {
	UserID          uint
	TargetUserID    uint
	CanViewCalendar bool
}

type RelationPermissionResult struct {
	UserID          uint `json:"user_id"`
	TargetUserID    uint `json:"target_user_id"`
	CanViewCalendar bool `json:"can_view_calendar"`
}

func NewRelationService(userRelationDAO dao.UserRelationDAO, cacheStore *cachepkg.Store) *RelationService {
	return &RelationService{
		userRelationDAO: userRelationDAO,
		cacheStore:      cacheStore,
	}
}

func (s *RelationService) SetCalendarPermission(ctx context.Context, input SetRelationPermissionInput) (*RelationPermissionResult, error) {
	if input.TargetUserID == 0 || input.UserID == input.TargetUserID {
		return nil, ErrInvalidTargetUser
	}

	relation, err := s.userRelationDAO.SetCalendarPermission(ctx, input.UserID, input.TargetUserID, input.CanViewCalendar)
	if err != nil {
		return nil, err
	}
	if s.cacheStore != nil {
		_ = s.cacheStore.DeleteByPattern(ctx, cachepkg.CalendarViewerOwnerPattern(input.UserID, input.TargetUserID))
	}

	return &RelationPermissionResult{
		UserID:          relation.UserID,
		TargetUserID:    relation.TargetUserID,
		CanViewCalendar: relation.CanViewCalendar,
	}, nil
}
