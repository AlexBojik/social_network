package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	SessionKey           string
	SessionName          string
	Database             string
	DatabaseUser         string
	DatabasePassword     string
	MaxOpenConnections   int
	DatabaseMasterServer string
	DatabaseSlaveServers []string
}

func NewConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	return &Config{
		SessionKey:           getEnv("SESSION_KEY", ""),
		SessionName:          getEnv("SESSION_NAME", ""),
		Database:             getEnv("DATABASE", ""),
		DatabaseUser:         getEnv("DATABASE_USER", ""),
		DatabasePassword:     getEnv("DATABASE_PASSWORD", ""),
		MaxOpenConnections:   getEnvAsInt("MAX_OPEN_CONNECTIONS", 60),
		DatabaseSlaveServers: getEnvAsSlice("DATABASE_SLAVE_SERVERS", []string{}, ","),
		DatabaseMasterServer: getEnv("DATABASE_MASTER_SERVER", ""),
	}
}

func getEnvAsSlice(name string, defaultVal []string, sep string) []string {
	valStr := getEnv(name, "")

	if valStr == "" {
		return defaultVal
	}

	val := strings.Split(valStr, sep)

	return val
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

func getEnv(key string, defaultVal string) string {
	if value, exist := os.LookupEnv(key); exist {
		return value
	}

	return defaultVal
}
