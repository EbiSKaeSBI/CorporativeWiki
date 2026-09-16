package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port             string
	DatabaseHost     string
	DatabaseUser     string
	DatabasePort     int
	DatabasePassword string
	DatabaseName     string
	JwtSecret        string
	JwtAccessTTL     time.Duration
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() *Config {
	portEnv := os.Getenv("PORT")
	dbPort, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	if dbPort == 0 {
		dbPort = 5433
	}
	jwtAccessTTL, _ := time.ParseDuration(os.Getenv("JWT_ACCESS_TTL"))
	if jwtAccessTTL == 0 {
		jwtAccessTTL = 24 * time.Hour
	}

	var port string
	if portEnv == "" {
		port = ":8080"
	} else {
		port = ":" + portEnv
	}
	return &Config{
		Port:             port,
		DatabaseHost:     getEnv("DB_HOST", "localhost"),
		DatabaseUser:     getEnv("DB_USER", "wiki"),
		DatabasePort:     dbPort,
		DatabasePassword: getEnv("DB_PASSWORD", "wiki"),
		DatabaseName:     getEnv("DB_NAME", "wiki"),
		JwtSecret:        getEnv("JWT_SECRET", "dev-secret"),
		JwtAccessTTL:     jwtAccessTTL,
	}
}
