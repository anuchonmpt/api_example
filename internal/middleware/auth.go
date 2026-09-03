package middleware

import (
	"strconv"

	"github.com/example/api-example/internal/auth"
	"github.com/example/api-example/internal/domain"
	apperrors "github.com/example/api-example/pkg/errors"
	"github.com/gin-gonic/gin"
)

const ActorContextKey = "actor"

func Auth(manager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := auth.ExtractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			apperrors.AbortGin(c, apperrors.ErrAuthInvalidTokenFormat)
			return
		}
		claims, err := manager.ValidateAccessToken(token)
		if err != nil {
			apperrors.AbortGin(c, apperrors.ErrAuthInvalidToken)
			return
		}
		userID, err := strconv.ParseInt(claims.UserID, 10, 64)
		if err != nil || userID <= 0 {
			apperrors.AbortGin(c, apperrors.ErrAuthInvalidToken)
			return
		}
		c.Set(ActorContextKey, domain.Actor{UserID: userID, Role: domain.Role(claims.Role)})
		c.Next()
	}
}

func GetActor(c *gin.Context) (domain.Actor, bool) {
	value, exists := c.Get(ActorContextKey)
	if !exists {
		return domain.Actor{}, false
	}
	actor, ok := value.(domain.Actor)
	return actor, ok
}
