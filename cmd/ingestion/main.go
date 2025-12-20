package main

import (
	"context"
	"log"

	"github.com/MKVAppDev/go-ingestion/internal/config"
	"github.com/MKVAppDev/go-ingestion/internal/dnse"
	auth "github.com/MKVAppDev/go-ingestion/internal/dnseauth"
	"github.com/MKVAppDev/go-ingestion/internal/redispub"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// Check Redis connection before starting
	pub := redispub.New(cfg.RedisAddr)
	defer pub.Close()

	if err := pub.Ping(ctx); err != nil {
		log.Fatalf("redis connection failed: %v (addr: %s)", err, cfg.RedisAddr)
	}
	log.Printf("redis connected successfully at %s", cfg.RedisAddr)

	token, err := auth.Authentication(ctx, cfg.Username, cfg.Password)
	if err != nil {
		log.Fatalf("authenticate failed: %v", err)
	}

	info, err := auth.GetInvestorInfo(ctx, token)
	if err != nil {
		log.Fatalf("getInvestorInfo failed: %v", err)
	}

	// 4 workers with 50000 buffer
	client := dnse.NewClient(pub, cfg.Env, 4, 50000)

	err = client.Run(info.InvestorID, token)

	if err != nil {
		log.Fatalf("dnse client error: %v", err)
	}
}
