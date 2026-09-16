package main

import (
	"context"
	"log"
	"net/http"

	"backend/internal/broker"
	"backend/internal/config"
	"backend/internal/handler"
	"backend/internal/repository"
	"backend/internal/service"
	"backend/internal/swagger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.LoadConfig()

	port := cfg.Port
	if port == "" {
		port = "8081"
	}

	dbpool, err := pgxpool.New(context.Background(), cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbpool.Close()

	userRepo := repository.NewUserRepository(dbpool)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authService)

	userService := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(userService, authService)

	dealRepo := repository.NewDealRepository(dbpool)
	discountPolicyRepo := repository.NewDiscountPolicyRepository(dbpool)
	discountPolicyHandler := handler.NewDiscountPolicyHandler(discountPolicyRepo)
	ancillaryUnitRepo := repository.NewAncillaryUnitRepository(dbpool)
	ancillaryUnitHandler := handler.NewAncillaryUnitHandler(ancillaryUnitRepo)
	erpRepo := repository.NewERPRepository(dbpool)
	erpHandler := handler.NewERPHandler(erpRepo)
	reminderRepo := repository.NewReminderRepository(dbpool)
	reminderHandler := handler.NewReminderHandler(reminderRepo)
	dealService := service.NewDealService(dealRepo, cfg.MaxManagerDiscountPercent, cfg.MaxSupervisorDiscountPercent, discountPolicyRepo)
	dealHandler := handler.NewDealHandler(dealService)

	emailSender := service.NewSMTPEmailSender(cfg)
	notificationRepo := repository.NewNotificationRepository(dbpool)
	notificationService := service.NewNotificationService(notificationRepo, dealRepo, userRepo, emailSender)

	constructionRepo := repository.NewConstructionRepository(dbpool)
	constructionService := service.NewConstructionService(constructionRepo, notificationService)
	constructionHandler := handler.NewConstructionHandler(constructionService)

	chatRepo := repository.NewChatRepository(dbpool)
	chatService := service.NewChatService(chatRepo)
	chatHandler := handler.NewChatHandler(chatService)

	dealService.SetNotificationDependencies(notificationRepo, userRepo, chatRepo, emailSender)

	dialogAnalysisService := service.NewDialogAnalysisService(dealRepo, chatRepo)
	competitorRepo := repository.NewCompetitorRepository(dbpool)
	competitorHandler := handler.NewCompetitorHandler(competitorRepo)
	recommendationRepo := repository.NewRecommendationRepository(dbpool)
	offerRepo := repository.NewOfferRepository(dbpool)
	offerService, err := service.NewOfferService(
		dealRepo,
		userRepo,
		offerRepo,
		cfg.MaxManagerDiscountPercent,
		cfg.MaxSupervisorDiscountPercent,
		discountPolicyRepo,
	)
	if err != nil {
		log.Fatalf("Invalid offer discount policy: %v\n", err)
	}
	offerDeliveryRepo := repository.NewOfferDeliveryRepository(dbpool)
	offerDeliveryService := service.NewOfferDeliveryService(offerDeliveryRepo, dealRepo, userRepo, service.NewSMTPOfferAttachmentSender(cfg))
	offerHandler := handler.NewOfferHandler(offerService, offerRepo, offerDeliveryService)

	aiService := service.NewAIAgentService(cfg.RabbitMQURL)
	defer aiService.Close()
	aiAuditRepo := repository.NewAIAuditRepository(dbpool)
	aiHandler := handler.NewAIHandler(aiService, chatService, aiAuditRepo)
	dialogAIHandler := handler.NewDialogAIHandler(aiService, dealService, aiAuditRepo)

	backendRPCDispatcher := broker.NewBackendRPCDispatcher(
		dealService,
		userService,
		constructionService,
		dialogAnalysisService,
		userService,
		constructionService,
		dealService,
		competitorRepo,
		recommendationRepo,
		offerService,
		erpRepo,
	)
	backendRPCConsumer := broker.NewBackendRPCConsumer(aiService, backendRPCDispatcher)
	if err := backendRPCConsumer.Start(context.Background()); err != nil {
		log.Fatalf("Unable to start Backend RPC consumer: %v\n", err)
	}
	defer backendRPCConsumer.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", swagger.Handler(swagger.StaffSpec, "Staff Server API - Swagger UI"))

	r.Post("/api/auth/login", authHandler.LoginStaff)
	r.Post("/api/auth/logout", authHandler.Logout)

	r.Group(func(r chi.Router) {
		r.Use(authHandler.AuthMiddleware)
		r.Use(authHandler.RequireStaffRole)

		r.Get("/api/staff/profile", authHandler.Profile)
		r.With(authHandler.RequireSupervisorRole).Post("/api/auth/register", authHandler.CreateStaff)
		r.Get("/api/users", userHandler.GetUsers)
		r.Get("/api/users/{id}", userHandler.GetUser)

		r.Get("/api/chat/sessions", chatHandler.GetAllSessions)
		r.Get("/api/chat/sessions/{id}", chatHandler.GetSession)
		r.Post("/api/chat/sessions/{id}/take", chatHandler.TakeSession)
		r.Post("/api/chat/sessions/{id}/close", chatHandler.CloseSession)
		r.Post("/api/chat/sessions/{id}/reject", chatHandler.RejectSession)
		r.Get("/api/chat/sessions/{id}/messages", chatHandler.GetMessages)
		r.Post("/api/chat/sessions/{id}/messages", chatHandler.SendMessage)

		r.Post("/api/deals", dealHandler.CreateDeal)
		r.Get("/api/deals", dealHandler.GetAllDeals)
		r.Get("/api/deals/{id}", dealHandler.GetDeal)
		r.Put("/api/deals/{id}/status", dealHandler.UpdateDealStatus)

		r.Get("/api/offers", offerHandler.List)
		r.Get("/api/offers/{id}", offerHandler.Get)
		r.Post("/api/offers/calculate", offerHandler.Calculate)
		r.Post("/api/offers", offerHandler.Create)
		r.Post("/api/offers/{id}/approval-request", offerHandler.RequestApproval)
		r.With(authHandler.RequireSupervisorRole).Post("/api/offers/{id}/approve", offerHandler.Approve)
		r.With(authHandler.RequireSupervisorRole).Post("/api/offers/{id}/reject", offerHandler.Reject)
		r.Get("/api/offers/{id}/pdf", offerHandler.PDF)
		r.Post("/api/offers/{id}/send", offerHandler.Send)

		r.Post("/api/ai/chat", aiHandler.Chat)
		r.Post("/api/ai/dialog/analyze", dialogAIHandler.Analyze)
		r.Post("/api/ai/dialog/reply-assist", dialogAIHandler.ReplyAssist)

		r.Get("/api/competitors", competitorHandler.List)
		r.With(authHandler.RequireSupervisorRole).Post("/api/competitors", competitorHandler.Create)
		r.With(authHandler.RequireSupervisorRole).Put("/api/competitors/{id}", competitorHandler.Update)

		r.Post("/api/complexes", constructionHandler.CreateComplex)
		r.Get("/api/complexes", constructionHandler.GetAllComplexes)
		r.Get("/api/complexes/{id}", constructionHandler.GetComplex)
		r.With(authHandler.RequireSupervisorRole).Put("/api/complexes/{id}", constructionHandler.UpdateComplex)
		r.Get("/api/complexes/{complexId}/buildings", constructionHandler.GetBuildingsByComplex)

		r.Post("/api/buildings", constructionHandler.CreateBuilding)
		r.Get("/api/buildings/{id}", constructionHandler.GetBuilding)
		r.With(authHandler.RequireSupervisorRole).Put("/api/buildings/{id}", constructionHandler.UpdateBuilding)
		r.Get("/api/buildings/{buildingId}/apartments", constructionHandler.GetApartmentsByBuilding)
		r.Get("/api/buildings/{buildingId}/progress", constructionHandler.GetProgressByBuilding)

		r.Post("/api/apartments", constructionHandler.CreateApartment)
		r.Get("/api/apartments/{id}", constructionHandler.GetApartment)
		r.With(authHandler.RequireSupervisorRole).Put("/api/apartments/{id}", constructionHandler.UpdateApartment)

		r.Post("/api/progress", constructionHandler.CreateProgress)
		r.Get("/api/progress/{id}", constructionHandler.GetProgress)
		r.With(authHandler.RequireSupervisorRole).Put("/api/progress/{id}", constructionHandler.UpdateProgress)

		r.Get("/api/buildings/{buildingId}/discount-policies", discountPolicyHandler.List)
		r.With(authHandler.RequireSupervisorRole).Post("/api/discount-policies", discountPolicyHandler.Create)
		r.Get("/api/buildings/{buildingId}/ancillary-units", ancillaryUnitHandler.List)
		r.Post("/api/ancillary-units", ancillaryUnitHandler.Create)
		r.With(authHandler.RequireSupervisorRole).Put("/api/ancillary-units/{id}", ancillaryUnitHandler.Update)
		r.Get("/api/buildings/{buildingId}/erp", erpHandler.List)
		r.Post("/api/erp/events", erpHandler.CreateEvent)
		r.Post("/api/erp/material-stocks", erpHandler.UpsertStock)
		r.Post("/api/erp/production-schedules", erpHandler.CreateSchedule)
		r.Get("/api/reminders", reminderHandler.List)
		r.Post("/api/reminders", reminderHandler.Create)
		r.Put("/api/reminders/{id}/complete", reminderHandler.Complete)
	})

	log.Printf("Starting Staff Server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
