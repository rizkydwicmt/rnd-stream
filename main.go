package main

import (
	"context"
	"fmt"
	"stream/application/health"
	"stream/application/tickets"
	"stream/common"

	"log"
	"net/http"
	"runtime"
	"runtime/debug"
	"stream/middleware"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	runtime.GOMAXPROCS(1)
	fmt.Printf("⚙️  CPU limit set to: %d core(s) (GOMAXPROCS=%d)\n", 1, runtime.GOMAXPROCS(0))

	// Set memory limit to 128 MB
	memLimit := int64(128 * 1024 * 1024) // 128 MB in bytes
	debug.SetMemoryLimit(memLimit)

	// Start
	db, err := setupDatabase()
	if err != nil {
		log.Fatal("Failed to setup database:", err)
	}
	z := NewLogger()
	r := SetupRouter(db)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  55 * time.Second,
		WriteTimeout: 55 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	memMonitorDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				z.Info("📊 Resource Monitor",
					zap.Uint64("alloc_mb", m.Alloc/(1024*1024)),
					zap.Uint64("sys_mb", m.Sys/(1024*1024)),
					zap.Uint32("gc_count", m.NumGC),
					zap.Int("goroutines", runtime.NumGoroutine()),
					zap.Int("cpu_cores", runtime.GOMAXPROCS(0)),
					zap.Int("num_cpu", runtime.NumCPU()),
				)
			case <-memMonitorDone:
				return
			}
		}
	}()

	go func() {
		log.Println("🚀 Server starting on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed:", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("🛑 Shutting down server...")
	srv.Shutdown(context.Background())
}

func NewLogger() *zap.Logger {
	var zapLogger *zap.Logger
	var err error

	zapLogger, err = zap.NewDevelopment()

	if err != nil {
		panic(err)
	}

	return zapLogger
}

func setupDatabase() (*gorm.DB, error) {
	// Open SQLite in-memory database for demo
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&common.Ticket{}); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	// Seed data (100,000 tickets for realistic demo)
	log.Println("🌱 Seeding database with 100,000 tickets...")
	if err := seedData(db); err != nil {
		return nil, fmt.Errorf("failed to seed data: %w", err)
	}
	log.Println("✅ Database seeded successfully")

	return db, nil
}

func seedData(db *gorm.DB) error {
	// Create tickets in batches for better performance
	const batchSize = 1000
	const totalTickets = 100000

	statuses := []string{"open", "in_progress", "pending", "resolved", "closed"}
	priorities := []string{"low", "medium", "high", "urgent"}

	for i := 0; i < totalTickets; i += batchSize {
		tickets := make([]common.Ticket, 0, batchSize)

		for j := 0; j < batchSize && i+j < totalTickets; j++ {
			id := i + j + 1
			tickets = append(tickets, common.Ticket{
				TicketNo:    fmt.Sprintf("TKT-%06d", id),
				CustomerID:  uint((id % 1000) + 1),
				Subject:     fmt.Sprintf("Issue #%d - Sample ticket", id),
				Description: fmt.Sprintf("This is a sample ticket description for ticket number %d", id),
				Status:      statuses[id%len(statuses)],
				Priority:    priorities[id%len(priorities)],
				CreatedAt:   time.Now().Add(-time.Duration(id) * time.Minute),
				UpdatedAt:   time.Now().Add(-time.Duration(id/2) * time.Minute),
			})
		}

		if err := db.Create(&tickets).Error; err != nil {
			return err
		}
	}

	return nil
}

func SetupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestInit())
	r.Use(middleware.ResponseInit())

	// Health endpoint
	healthRepo := health.NewRepository(db)
	healthSvc := health.NewService(healthRepo)
	healthHandler := health.NewHandler(healthSvc)

	// Tickets streaming endpoint
	ticketsRepo := tickets.NewRepository(db)
	ticketsSvc := tickets.NewService(ticketsRepo)
	ticketsHandler := tickets.NewHandler(ticketsSvc)

	api := r.Group("")
	healthHandler.RegisterRoutes(api)
	ticketsHandler.RegisterRoutes(api)

	return r
}
