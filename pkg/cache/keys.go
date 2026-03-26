package cache

import (
	"fmt"
	"time"
)

const keyPrefix = "ss:v1"

func EventListKey(ownerID uint, startFrom, startTo *time.Time, view string, date *time.Time, includeCancelled bool) string {
	includeFlag := "0"
	if includeCancelled {
		includeFlag = "1"
	}

	return fmt.Sprintf(
		"%s:events:list:owner:%d:from:%s:to:%s:view:%s:date:%s:inc_cancelled:%s",
		keyPrefix,
		ownerID,
		timeOrPlaceholder(startFrom, false),
		timeOrPlaceholder(startTo, false),
		stringOrPlaceholder(view),
		timeOrPlaceholder(date, true),
		includeFlag,
	)
}

func EventItemKey(ownerID, eventID uint) string {
	return fmt.Sprintf("%s:events:item:owner:%d:id:%d", keyPrefix, ownerID, eventID)
}

func EventListPattern(ownerID uint) string {
	return fmt.Sprintf("%s:events:list:owner:%d:*", keyPrefix, ownerID)
}

func CalendarKey(viewerID, ownerID uint, view string, date time.Time) string {
	return fmt.Sprintf(
		"%s:calendar:viewer:%d:owner:%d:view:%s:date:%s",
		keyPrefix,
		viewerID,
		ownerID,
		stringOrPlaceholder(view),
		date.Format("2006-01-02"),
	)
}

func CalendarOwnerPattern(ownerID uint) string {
	return fmt.Sprintf("%s:calendar:*:owner:%d:*", keyPrefix, ownerID)
}

func CalendarViewerOwnerPattern(viewerID, ownerID uint) string {
	return fmt.Sprintf("%s:calendar:viewer:%d:owner:%d:*", keyPrefix, viewerID, ownerID)
}

func timeOrPlaceholder(t *time.Time, dateOnly bool) string {
	if t == nil {
		return "_"
	}
	if dateOnly {
		return t.Format("2006-01-02")
	}
	return t.UTC().Format(time.RFC3339)
}

func stringOrPlaceholder(s string) string {
	if s == "" {
		return "_"
	}
	return s
}
