package main

import (
	"context"
	"fmt"
	"os"
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
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	runtime.GOMAXPROCS(1)
	fmt.Printf("⚙️  CPU limit set to: %d core(s) (GOMAXPROCS=%d)\n", 1, runtime.GOMAXPROCS(0))

	// Set memory limit to 128 MB
	memLimit := int64(128 * 1024 * 1024) // 128 MB in bytes
	debug.SetMemoryLimit(memLimit)

	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using environment variables")
	}

	// Setup dummy database (SQLite in-memory)
	dummyDB, err := setupDummyDatabase()
	if err != nil {
		log.Fatal("Failed to setup dummy database:", err)
	}

	// Setup real database (MySQL)
	realDB, err := setupRealDatabase()
	if err != nil {
		log.Fatal("Failed to setup real database:", err)
	}

	z := NewLogger()
	r := SetupRouter(dummyDB, realDB)

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

func setupDummyDatabase() (*gorm.DB, error) {
	log.Println("📦 Setting up dummy database (SQLite in-memory)...")
	// Open SQLite in-memory database for demo
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect dummy database: %w", err)
	}

	// Auto-migrate
	if err := db.AutoMigrate(&common.Ticket{}); err != nil {
		return nil, fmt.Errorf("failed to migrate dummy database: %w", err)
	}

	// Seed data (100,000 tickets for realistic demo)
	log.Println("🌱 Seeding dummy database with 100,000 tickets...")
	if err := seedData(db); err != nil {
		return nil, fmt.Errorf("failed to seed dummy data: %w", err)
	}
	log.Println("✅ Dummy database seeded successfully")

	return db, nil
}

func setupRealDatabase() (*gorm.DB, error) {
	log.Println("🗄️  Setting up real database (MySQL)...")

	// Get environment variables
	host := os.Getenv("REAL_DB_HOST")
	port := os.Getenv("REAL_DB_PORT")
	user := os.Getenv("REAL_DB_USER")
	pass := os.Getenv("REAL_DB_PASS")
	dbname := os.Getenv("REAL_DB_NAME")

	// Validate required environment variables
	if host == "" || port == "" || user == "" || pass == "" || dbname == "" {
		return nil, fmt.Errorf("missing required real database environment variables")
	}

	// Build MySQL DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, host, port, dbname)

	// Open MySQL database connection
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect real database: %w", err)
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping real database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("✅ Real database connected successfully")

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

func SetupRouter(dummyDB *gorm.DB, realDB *gorm.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestInit())
	r.Use(middleware.ResponseInit())

	// Health endpoint (monitors both databases)
	dummyHealthRepo := health.NewRepository(dummyDB)
	realHealthRepo := health.NewRepository(realDB)
	healthSvc := health.NewService(dummyHealthRepo, realHealthRepo)
	healthHandler := health.NewHandler(healthSvc)

	// Dummy database tickets streaming endpoint
	dummyTicketsRepo := tickets.NewRepository(dummyDB)
	dummyTicketsSvc := tickets.NewService(dummyTicketsRepo)
	dummyTicketsHandler := tickets.NewHandler(dummyTicketsSvc)

	// Real database tickets streaming endpoint
	realTicketsRepo := tickets.NewRepository(realDB)
	realTicketsSvc := tickets.NewService(realTicketsRepo)
	realTicketsHandler := tickets.NewHandler(realTicketsSvc)

	// Register routes
	api := r.Group("")
	healthHandler.RegisterRoutes(api)

	// Register dummy database routes under /v1/tickets
	dummyGroup := api.Group("/v1/tickets")
	dummyTicketsHandler.RegisterRoutesWithPrefix(dummyGroup)

	// Register real database routes under /v1/tickets-real
	realGroup := api.Group("/v1/tickets-real")
	realTicketsHandler.RegisterRoutesWithPrefix(realGroup)

	return r
}
