package commands

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"nos3"
	"nos3/config"
	"nos3/internal/application/usecase"
	"nos3/internal/infrastructure/broker"
	"nos3/internal/infrastructure/database"
	"nos3/internal/infrastructure/grpcclient"
	"nos3/internal/infrastructure/minio"
	"nos3/internal/presentation/handler"
	"nos3/internal/presentation/middleware"
	"nos3/pkg/logger"
)

func HandleRun(args []string) {
	if len(args) < 3 {
		ExitOnError(errors.New("at least 1 arguments expected\nuse help command for more information"))
	}

	cfg, err := config.Load(args[2])
	if err != nil {
		ExitOnError(err)
	}

	logger.InitGlobalLogger(&cfg.Logger)

	logger.Info("running nos3", "version", nos3.StringVersion())

	grpcClient, err := grpcclient.New(cfg.GRPCClient)
	if err != nil {
		ExitOnError(err)
	}

	resp, err := grpcClient.RegisterService(context.Background(), fmt.Sprint(cfg.GRPCServer.Port),
		cfg.GRPCClient.Region)
	if err != nil {
		ExitOnError(err)
	}

	if !resp.Success {
		ExitOnError(fmt.Errorf("cant register to master: %s", *resp.Message))
	}

	brokerClient, err := broker.NewClient(cfg.BrokerConfig, grpcClient)
	if err != nil {
		ExitOnError(err)
	}

	brokerPublisher := broker.NewPublisher(brokerClient, cfg.PublisherConfig, grpcClient)

	db, err := database.Connect(cfg.DBConfig, grpcClient)
	if err != nil {
		ExitOnError(err)
	}

	dbRemover := database.NewRemover(db, grpcClient)
	dbRetriever := database.NewBlobRetriever(db, grpcClient)
	dbWriter := database.NewBlobWriter(db, grpcClient)

	minIOClient, err := minio.New(cfg.MinIOClient, grpcClient)
	if err != nil {
		ExitOnError(err)
	}
	minIORemover := minio.NewRemover(minIOClient.MinioClient, grpcClient, cfg.MinIORemover)
	minIOUploader := minio.NewUploader(minIOClient.MinioClient, grpcClient, cfg.MinIOUploader)

	uploader := usecase.NewUploader(brokerPublisher, dbRetriever, dbWriter, minIOUploader,
		minIORemover, dbRemover, cfg.Default.Address)

	uploadHandler := handler.NewUploadHandler(uploader)
	e := echo.New()
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderContentLength},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost,
			http.MethodDelete, http.MethodHead, http.MethodOptions},
		MaxAge: 86400,
	}))
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.Secure())
	e.Use(echoMiddleware.BodyLimit("50M"))
	e.Use(echoMiddleware.RateLimiter(echoMiddleware.NewRateLimiterMemoryStore(20)))

	e.GET("/health", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	e.POST("/", uploadHandler.Handle, middleware.AuthMiddleware("upload"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := e.Start(cfg.Default.Address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ExitOnError(fmt.Errorf("shutting down server: %w", err))
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		ExitOnError(err)
	}
}
