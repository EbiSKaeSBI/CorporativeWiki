package config

import "os"

type Config struct {
	Port             string
	DatabaseHost     string
	DatabaseUser     string
	DatabasePort     int
	DatabasePassword string
	DatabaseName     string
}

func Load() *Config {
	portEnv := os.Getenv("PORT")
	var port string
	if portEnv == "" {
		port = ":8080"
	} else {
		port = ":" + portEnv
	}
	DB_HOST := "localhost"
	DB_PORT := 5433
	DB_USER := "wiki"
	DB_PASSWORD := "wiki"
	DB_NAME := "wiki"
	return &Config{
		Port: port,
		DatabaseHost: DB_HOST,
		DatabaseUser: DB_USER,
		DatabasePort: DB_PORT,
		DatabasePassword: DB_PASSWORD,
		DatabaseName: DB_NAME,
	}
}
