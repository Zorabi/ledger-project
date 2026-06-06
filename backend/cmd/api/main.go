package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "ledger-backend/docs"
	"ledger-backend/internal/config"
	"ledger-backend/internal/database"
	"ledger-backend/internal/handlers"
	applogger "ledger-backend/internal/logger"
	"ledger-backend/internal/middleware"
	"ledger-backend/internal/scheduler"
	"ledger-backend/internal/utils"
)

// @title 礼尚往来 API
// @version 1.5.0.0
// @description 家庭租户礼尚往来记录系统后端接口。
// @host localhost:8080
// @BasePath /api/v1
func main() {
	config.LoadConfig()
	logOutput, err := applogger.Setup(config.AppConfig.Log)
	if err != nil {
		log.Fatalf("setup logger: %s", err)
	}
	defer logOutput.Close()
	slog.Info("logger initialized",
		"level", config.AppConfig.Log.Level,
		"console", config.AppConfig.Log.Console,
		"file", config.AppConfig.Log.File,
		"file_path", config.AppConfig.Log.FilePath,
	)

	db := database.ConnectDatabase()

	app := handlers.NewAppHandler(db)
	if err := app.SeedDefaults(); err != nil {
		log.Fatalf("seed defaults: %s", err)
	}

	ctx, stopScheduler := context.WithCancel(context.Background())
	defer stopScheduler()
	scheduler.New(db).Start(ctx)

	r := gin.New()
	r.Use(middleware.StructuredLogger())
	r.Use(utils.Recovery())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Source-File"},
	}))

	app.RegisterRoutes(r)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.AppConfig.Server.Port),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("server shutdown:", err)
	}
}
