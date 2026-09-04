package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/example/api-example/internal/auth"
	"github.com/example/api-example/internal/config"
	"github.com/example/api-example/internal/domain"
	"github.com/example/api-example/internal/handler"
	queueadapter "github.com/example/api-example/internal/queue"
	"github.com/example/api-example/internal/repository"
	"github.com/example/api-example/internal/router"
	"github.com/example/api-example/internal/service"
	storageadapter "github.com/example/api-example/internal/storage"
	"github.com/example/api-example/pkg/database"
	"github.com/sirupsen/logrus"
)

type Runtime struct {
	Router http.Handler
	Worker *service.DocumentWorker
	Queue  domain.DocumentQueue
	stack  *resourceStack
}

func Build(ctx context.Context, cfg *config.Config) (*Runtime, error) {
	stack := &resourceStack{}
	fail := func(err error) (*Runtime, error) { return nil, errors.Join(err, stack.Close()) }
	pool, err := database.NewPostgres(ctx, cfg.Database.URL(), cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns, cfg.Database.ConnMaxLifetime)
	if err != nil {
		return fail(err)
	}
	stack.Add(func() error { pool.Close(); return nil })
	redisClient, err := database.NewRedis(ctx, cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		return fail(err)
	}
	stack.Add(redisClient.Close)
	objectStorage, err := storageadapter.New(ctx, cfg.Storage)
	if err != nil {
		return fail(err)
	}
	queue := queueadapter.NewRedis(redisClient, cfg.Redis.DocumentQueue, cfg.Redis.ProcessingQueue, cfg.Redis.DeadLetterQueue, cfg.Redis.DocumentMaxAttempts)
	authRepository := repository.NewAuthRepository(pool)
	documentRepository := repository.NewDocumentRepository(pool)
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.Audience, cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)
	authService := service.NewAuthService(authRepository, jwtManager)
	documentService := service.NewDocumentService(documentRepository, objectStorage, queue, cfg.Storage.MaxUploadBytes, cfg.Storage.AllowedMediaTypes, cfg.Storage.CDNURL)
	engine := router.New(router.Config{JWT: jwtManager, AuthHandler: handler.NewAuth(authService), DocumentHandler: handler.NewDocument(documentService), AllowedOrigins: cfg.CORS.AllowedOrigins})
	return &Runtime{Router: engine, Worker: service.NewDocumentWorker(documentRepository, objectStorage), Queue: queue, stack: stack}, nil
}

func (r *Runtime) Close() error { return r.stack.Close() }

type resourceStack struct {
	mu      sync.Mutex
	once    sync.Once
	closers []func() error
	err     error
}

func (s *resourceStack) Add(close func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closers = append(s.closers, close)
}
func (s *resourceStack) Close() error {
	s.once.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for index := len(s.closers) - 1; index >= 0; index-- {
			s.err = errors.Join(s.err, s.closers[index]())
		}
	})
	return s.err
}

func RunServer(ctx context.Context, cfg *config.Config, log *logrus.Logger) error {
	runtime, err := Build(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := runtime.Close(); closeErr != nil {
			log.WithError(closeErr).Error("close resources")
		}
	}()
	server := &http.Server{Addr: ":" + cfg.App.Port, Handler: runtime.Router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	errorsChannel := make(chan error, 1)
	go func() { errorsChannel <- server.ListenAndServe() }()
	log.WithField("address", server.Addr).Info("HTTP server started")
	select {
	case err := <-errorsChannel:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	}
}

func RunWorker(ctx context.Context, cfg *config.Config, log *logrus.Logger) error {
	runtime, err := Build(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := runtime.Close(); closeErr != nil {
			log.WithError(closeErr).Error("close resources")
		}
	}()
	log.Info("document worker started")
	err = runtime.Worker.Run(ctx, runtime.Queue)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
