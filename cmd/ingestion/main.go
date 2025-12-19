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

	token, err := auth.Authentication(ctx, cfg.Username, cfg.Password)
	if err != nil {
		log.Fatalf("authenticate failed: %v", err)
	}

	info, err := auth.GetInvestorInfo(ctx, token)
	if err != nil {
		log.Fatalf("getInvestorInfo failed: %v", err)
	}

	pub := redispub.New(cfg.RedisAddr)
	defer pub.Close()

	tickers := []string{"MSB", "FPT"}

	// 4 workers with 50000 buffer
	client := dnse.NewClient(pub, cfg.Env, 4, 50000)

	err = client.Run(info.InvestorID, token, tickers)

	if err != nil {
		log.Fatalf("dnse client error: %v", err)
	}
}
