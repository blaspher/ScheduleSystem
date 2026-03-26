package api

import (
	"net/http"

	jwtpkg "schedule-system/pkg/jwt"
	"schedule-system/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	authHandler *AuthHandler,
	eventHandler *EventHandler,
	meetingHandler *MeetingHandler,
	relationHandler *RelationHandler,
	calendarHandler *CalendarHandler,
	jwtMgr *jwtpkg.Manager,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)

	protected := v1.Group("")
	protected.Use(middleware.JWTAuth(jwtMgr))
	protected.GET("/me", authHandler.Me)
	protected.POST("/relations", relationHandler.SetCalendarPermission)
	protected.GET("/users/:id/calendar", calendarHandler.GetUserCalendar)
	protected.POST("/meetings", meetingHandler.Create)
	protected.GET("/meetings/invitations", meetingHandler.ListInvitations)
	protected.POST("/meetings/:id/accept", meetingHandler.Accept)
	protected.POST("/meetings/:id/reject", meetingHandler.Reject)
	protected.POST("/events", eventHandler.Create)
	protected.PUT("/events/:id", eventHandler.Update)
	protected.DELETE("/events/:id", eventHandler.Delete)
	protected.GET("/events", eventHandler.List)
	protected.GET("/events/:id", eventHandler.GetByID)

	return r
}
