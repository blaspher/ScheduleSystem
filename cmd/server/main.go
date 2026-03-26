package main

import (
	"log"

	"schedule-system/config"
	"schedule-system/internal/api"
	"schedule-system/internal/dao"
	"schedule-system/internal/model"
	"schedule-system/internal/service"
	"schedule-system/pkg/cache"
	"schedule-system/pkg/database"
	jwtpkg "schedule-system/pkg/jwt"
)

func main() {
	log.Println("loading configuration")
	cfg := config.Load()

	log.Println("initializing mysql and bootstrapping database")
	db, err := database.NewMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("failed to connect mysql: %v", err)
	}
	log.Println("mysql ready")

	log.Println("running database migrations")
	if err := db.AutoMigrate(&model.User{}, &model.Event{}, &model.UserRelation{}, &model.MeetingAttendee{}); err != nil {
		log.Fatalf("failed to migrate schema: %v", err)
	}
	log.Println("database migrations completed")

	log.Println("connecting redis")
	redisClient, err := cache.NewRedis(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("redis ready")

	log.Println("initializing application components")
	jwtMgr := jwtpkg.NewManager(cfg.JWTSecret)
	cacheStore := cache.NewStore(redisClient, 0)

	userDAO := dao.NewUserDAO(db)
	eventDAO := dao.NewEventDAO(db)
	meetingDAO := dao.NewMeetingDAO(db)
	userRelationDAO := dao.NewUserRelationDAO(db)
	authService := service.NewAuthService(userDAO, jwtMgr)
	eventService := service.NewEventService(eventDAO, cacheStore)
	meetingService := service.NewMeetingService(meetingDAO)
	relationService := service.NewRelationService(userRelationDAO, cacheStore)
	calendarService := service.NewCalendarService(userRelationDAO, eventDAO, cacheStore)
	authHandler := api.NewAuthHandler(authService)
	eventHandler := api.NewEventHandler(eventService)
	meetingHandler := api.NewMeetingHandler(meetingService)
	relationHandler := api.NewRelationHandler(relationService)
	calendarHandler := api.NewCalendarHandler(calendarService)

	r := api.NewRouter(authHandler, eventHandler, meetingHandler, relationHandler, calendarHandler, jwtMgr)
	addr := ":" + cfg.ServerPort
	log.Printf("starting HTTP server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
