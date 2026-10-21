package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const verifiedTokenContextKey = "stylometry_verified_token"

// UserExists проверяет существование пользователя в PostgreSQL.
type UserExists func(uint) (bool, error)

// UserID возвращает проверенный ID из контекста или 0 для гостя.
func UserID(ctx *gin.Context) uint {
	token, exists := VerifiedTokenFromContext(ctx)
	if !exists {
		return 0
	}
	return token.UserID
}

// VerifiedTokenFromContext возвращает claims после проверки middleware.
func VerifiedTokenFromContext(ctx *gin.Context) (VerifiedToken, bool) {
	value, exists := ctx.Get(verifiedTokenContextKey)
	if !exists {
		return VerifiedToken{}, false
	}
	token, ok := value.(VerifiedToken)
	if !ok {
		return VerifiedToken{}, false
	}
	return token, true
}

// RequireJWT требует действительный Bearer JWT для операции создателя.
func RequireJWT(issuer *Issuer, userExists UserExists, blacklist Blacklist) gin.HandlerFunc {
	return verifyJWT(issuer, userExists, blacklist, true)
}

// OptionalJWT оставляет просмотр публичным, но учитывает действительный токен.
func OptionalJWT(issuer *Issuer, userExists UserExists, blacklist Blacklist) gin.HandlerFunc {
	return verifyJWT(issuer, userExists, blacklist, false)
}

func verifyJWT(issuer *Issuer, userExists UserExists, blacklist Blacklist, required bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.GetHeader("Authorization")
		if header == "" && !required {
			ctx.Next()
			return
		}
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "требуется действительный Bearer JWT"})
			return
		}
		token, err := issuer.Verify(parts[1])
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "требуется действительный Bearer JWT"})
			return
		}
		revoked, err := blacklist.IsRevoked(ctx.Request.Context(), token.JTI)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "проверка отзыва JWT временно недоступна"})
			return
		}
		if revoked {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "требуется действительный Bearer JWT"})
			return
		}
		exists, err := userExists(token.UserID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "ошибка базы данных"})
			return
		}
		if !exists {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "требуется действительный Bearer JWT"})
			return
		}
		ctx.Set(verifiedTokenContextKey, token)
		ctx.Next()
	}
}
