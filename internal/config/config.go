package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

type Config struct {
	Username  string
	Password  string
	RedisAddr string
	Env       string
}

func Load() *Config {

	envPath := LoadEnvPath()

	if err := godotenv.Load(envPath); err != nil {
		log.Printf("warning: cannot load .env from %s: %v", envPath, err)
	}

	username := os.Getenv("USR")
	password := os.Getenv("PASSWD")
	redisAddr := os.Getenv("REDIS_ADDR")
	env := os.Getenv("ENV")

	if username == "" {
		log.Fatal("username missing !")
	}

	if password == "" {
		log.Fatal("password missing !")
	}

	if redisAddr == "" {
		log.Fatal("redis is missing !")
	}

	if env == "" {
		log.Fatal("env is missing !")
	}

	return &Config{
		Username:  username,
		Password:  password,
		RedisAddr: redisAddr,
		Env:       env,
	}
}

func LoadEnvPath() string {

	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	envPath := filepath.Join(dir, "..", "..", ".env")

	return envPath
}
