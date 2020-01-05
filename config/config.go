package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	SessionKey       string
	SessionName      string
	Database         string
	DatabaseUser     string
	DatabasePassword string
	DatabaseServer   string
}

func NewConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	return &Config{
		SessionKey:       getEnv("SESSION_KEY", ""),
		SessionName:      getEnv("SESSION_NAME", ""),
		Database:         getEnv("DATABASE", ""),
		DatabaseUser:     getEnv("DATABASE_USER", ""),
		DatabasePassword: getEnv("DATABASE_PASSWORD", ""),
		DatabaseServer:   getEnv("DATABASE_SERVER", ""),
	}
}

func getEnv(key string, defaultVal string) string {
	if value, exist := os.LookupEnv(key); exist {
		return value
	}

	return defaultVal
}
