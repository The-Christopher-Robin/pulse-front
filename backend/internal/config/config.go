package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr         string
	GRPCAddr         string
	PostgresURL      string
	RedisAddr        string
	RedisPassword    string
	ExposureBuffer   int
	ExposureFlushDur time.Duration
	AllowedOrigin    string
	Environment      string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:         getenv("HTTP_ADDR", ":8080"),
		GRPCAddr:         getenv("GRPC_ADDR", ":9090"),
		PostgresURL:      getenv("POSTGRES_URL", "postgres://pulse:pulse@localhost:5432/pulse?sslmode=disable"),
		RedisAddr:        getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    os.Getenv("REDIS_PASSWORD"),
		AllowedOrigin:    getenv("ALLOWED_ORIGIN", "http://localhost:3000"),
		Environment:      getenv("ENVIRONMENT", "dev"),
		ExposureBuffer:   getenvInt("EXPOSURE_BUFFER", 512),
		ExposureFlushDur: getenvDuration("EXPOSURE_FLUSH_INTERVAL", 2*time.Second),
	}
	if cfg.ExposureBuffer <= 0 {
		return cfg, fmt.Errorf("EXPOSURE_BUFFER must be positive, got %d", cfg.ExposureBuffer)
	}
	return cfg, nil
}

func getenv(k, d string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return d
}

func getenvInt(k string, d int) int {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return d
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return n
}

func getenvDuration(k string, d time.Duration) time.Duration {
	v, ok := os.LookupEnv(k)
	if !ok || v == "" {
		return d
	}
	dur, err := time.ParseDuration(v)
	if err != nil {
		return d
	}
	return dur
}
