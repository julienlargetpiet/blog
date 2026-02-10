package config

import (
	"log"
	"os"
	"strconv"

	"blog/internal/db"
)

type Config struct {
	DB        db.Config
	AdminAddr string
    AdminPass string
}

func Load() Config {
	cfg := Config{
		DB: db.Config{
			User:     getEnv("BLOG_DB_USER", "blog_user"),
			Password: getEnv("BLOG_DB_PASSWORD", "password"),
			Host:     getEnv("BLOG_DB_HOST", "127.0.0.1"),
			Port:     getEnvInt("BLOG_DB_PORT", 3306),
			DBName:   getEnv("BLOG_DB_NAME", "go_blog"),
		},
		AdminAddr: getEnv("BLOG_ADMIN_ADDR", ":8080"),
		AdminPass: getEnv("BLOG_ADMIN_PASSWORD", "password"),
	}

	if cfg.DB.Password == "" {
		log.Fatal("BLOG_DB_PASSWORD must be set")
	}

	if cfg.AdminPass == "" {
		log.Fatal("BLOG_ADMIN_PASSWORD must be set")
	}

	return cfg
}

/* helpers */

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid %s: %v", key, err)
		}
		return i
	}
	return def
}


