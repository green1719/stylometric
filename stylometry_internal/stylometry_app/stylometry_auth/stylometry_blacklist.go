package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const BlacklistKeyPrefix = "jwt:blacklist:"

var ErrTokenExpired = errors.New("срок действия JWT истёк")

// Blacklist проверяет и сохраняет отзыв отдельных JWT.
type Blacklist interface {
	IsRevoked(context.Context, string) (bool, error)
	Revoke(context.Context, VerifiedToken) (bool, error)
}

type RedisBlacklist struct {
	client *redis.Client
}

// NewRedisBlacklistFromEnv проверяет конфигурацию и доступность Redis при запуске.
func NewRedisBlacklistFromEnv() (*RedisBlacklist, error) {
	address := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if address == "" {
		return nil, errors.New("REDIS_ADDR не задан")
	}
	database := 0
	if rawDatabase := strings.TrimSpace(os.Getenv("REDIS_DB")); rawDatabase != "" {
		parsedDatabase, err := strconv.Atoi(rawDatabase)
		if err != nil || parsedDatabase < 0 {
			return nil, errors.New("REDIS_DB должен быть неотрицательным числом")
		}
		database = parsedDatabase
	}
	client := redis.NewClient(&redis.Options{
		Addr:         address,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           database,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("Redis недоступен: %w", err)
	}
	return &RedisBlacklist{client: client}, nil
}

// IsRevoked ищет только случайный jti, не сохраняя полный JWT.
func (blacklist *RedisBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	count, err := blacklist.client.Exists(ctx, BlacklistKeyPrefix+jti).Result()
	return count > 0, err
}

// Revoke сохраняет ID пользователя с TTL до окончания действия JWT.
// SETNX не продлевает TTL уже отозванного токена.
func (blacklist *RedisBlacklist) Revoke(ctx context.Context, token VerifiedToken) (bool, error) {
	ttl := time.Until(token.ExpiresAt)
	if ttl <= 0 {
		return false, ErrTokenExpired
	}
	return blacklist.client.SetNX(
		ctx,
		BlacklistKeyPrefix+token.JTI,
		strconv.FormatUint(uint64(token.UserID), 10),
		ttl,
	).Result()
}
