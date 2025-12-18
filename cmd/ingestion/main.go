package main

import (
	"context"
	"log"

	"github.com/MKVAppDev/go-ingestion/internal/auth"
	"github.com/MKVAppDev/go-ingestion/internal/config"
	"github.com/MKVAppDev/go-ingestion/internal/dnse"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	token, err := auth.Authentication(ctx, cfg.Username, cfg.Password)
	if err != nil {
		log.Fatalf("authenticate failed: %v", err)
	}

	log.Printf("Got token len=%d", len(token))

	info, err := auth.GetInvestorInfo(ctx, token)
	if err != nil {
		log.Fatalf("getInvestorInfo failed: %v", err)
	}

	log.Printf("InvestorID: %s", info.InvestorID)

	if err := dnse.Run(info.InvestorID, token); err != nil {
		log.Fatalf("mqtt run failed: %v", err)
	}
}
