package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/internal/db/sqlc"
)

func RequireRoles(
	roles ...sqlc.UserRole,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		value, exists := c.Get(UserRoleKey)

		if !exists {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{
					"error": "permission denied",
				},
			)
			return
		}

		userRole, ok := value.(sqlc.UserRole)

		if !ok {
			c.AbortWithStatusJSON(
				http.StatusForbidden,
				gin.H{
					"error": "invalid user role",
				},
			)
			return
		}

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(
			http.StatusForbidden,
			gin.H{
				"error": "insufficient permissions",
			},
		)
	}
}
