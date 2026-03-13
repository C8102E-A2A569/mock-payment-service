// Точка входа приложения Mock Payment Service.
// Загрузка конфига, подключение к Postgres, миграции, сервис; далее — gRPC-сервер.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"new-project/internal/cache"
	"new-project/internal/config"
	grpcserver "new-project/internal/grpc"
	"new-project/internal/kafka"
	"new-project/internal/repository/postgres"
	"new-project/internal/service"
)

func main() {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "configs/config.yaml"
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = ""
	}
	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	dsn := cfg.DB.DSN()
	pool, err := postgres.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := postgres.RunMigrations(ctx, dsn); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	log.Printf("migrations applied")

	repo := postgres.NewAccountRepo(pool)
	var events service.EventProducer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.TopicPaymentEvents != "" {
		eventsProducer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicPaymentEvents)
		defer eventsProducer.Close()
		events = eventsProducer
		log.Printf("Kafka producer configured for topic %s", cfg.Kafka.TopicPaymentEvents)
	}
	var paymentCache service.PaymentCache
	if cfg.Redis.Addr != "" {
		redisCache, err := cache.NewRedisCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.BalanceTTL, cfg.Redis.IdempotencyTTL)
		if err != nil {
			log.Fatalf("redis connect: %v", err)
		}
		defer redisCache.Close()
		paymentCache = redisCache
		log.Printf("Redis cache configured at %s", cfg.Redis.Addr)
	}
	paymentSvc := service.NewPaymentService(repo, events, paymentCache)

	addr := fmt.Sprintf(":%d", cfg.Server.GRPCPort)
	lis, err := grpcserver.Listen(addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	s := grpcserver.NewServer(paymentSvc)
	log.Printf("gRPC server listening on %s", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
