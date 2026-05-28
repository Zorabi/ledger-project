package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ledger-backend/internal/models"
	"ledger-backend/internal/utils"
)

const AuthContextKey = "auth_user"

type AuthUser struct {
	UserID   uint
	FamilyID *uint
	Username string
	Role     string
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录或令牌缺失"})
			return
		}
		claims, err := utils.ParseToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "令牌无效或已过期"})
			return
		}
		c.Set(AuthContextKey, AuthUser{
			UserID:   claims.UserID,
			FamilyID: claims.FamilyID,
			Username: claims.Username,
			Role:     claims.Role,
		})
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		auth, ok := CurrentUser(c)
		if !ok || !allowed[auth.Role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无权访问该资源"})
			return
		}
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (AuthUser, bool) {
	value, ok := c.Get(AuthContextKey)
	if !ok {
		return AuthUser{}, false
	}
	auth, ok := value.(AuthUser)
	return auth, ok
}

func RequireFamily(c *gin.Context) (AuthUser, bool) {
	auth, ok := CurrentUser(c)
	if !ok || (auth.Role != models.RoleSuperAdmin && auth.FamilyID == nil) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "当前账号未归属家庭"})
		return AuthUser{}, false
	}
	return auth, true
}
