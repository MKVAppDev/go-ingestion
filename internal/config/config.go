package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Username string
	Password string
}

func Load() *Config {

	if err := godotenv.Load(".env"); err != nil {
		log.Printf("warning: cannot load ../.env: %v", err)
	}

	username := os.Getenv("usernameEntrade")
	password := os.Getenv("password")

	if username == "" {
		log.Fatal("username missing in env !")
	}

	if password == "" {
		log.Fatal("password missing in env !")
	}

	return &Config{
		Username: username,
		Password: password,
	}

}
