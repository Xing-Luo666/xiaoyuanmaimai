package config

import (
	"os"
	"strconv"
)

// Config UAS系统配置
type Config struct {
	ServerPort string
	DB         DBConfig
	JWT        JWTConfig
	OAuth2     OAuth2Config
	Redis      RedisConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
	Issuer      string
}

type OAuth2Config struct {
	CodeExpireSeconds  int // 授权码有效期（秒）
	TokenExpireSeconds int // Token有效期（秒）
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// Load 从环境变量加载配置
func Load() *Config {
	return &Config{
		ServerPort: getEnv("UAS_PORT", "8081"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", "114514"),
			DBName:   getEnv("DB_NAME", "uas_db"),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "uas-secret-key-2026-school-trade"),
			ExpireHours: getEnvInt("JWT_EXPIRE_HOURS", 24),
			Issuer:      "uas",
		},
		OAuth2: OAuth2Config{
			CodeExpireSeconds:  getEnvInt("OAUTH_CODE_EXPIRE", 300),
			TokenExpireSeconds: getEnvInt("OAUTH_TOKEN_EXPIRE", 604800),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
