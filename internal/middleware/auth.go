package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/internal/auth"
	"example.com/internal/db/sqlc"
)

const (
	UserIDKey         = "user_id"
	OrganizationIDKey = "organization_id"
	UserRoleKey       = "user_role"
	UserEmailKey      = "user_email"
)

func AuthMiddleware(
	queries *sqlc.Queries,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		token, err := c.Cookie("session")

		if err != nil || token == "" {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "authentication required",
				},
			)
			return
		}

		tokenHash := auth.HashToken(token)

		user, err := queries.GetSessionUser(
			c,
			tokenHash,
		)

		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				gin.H{
					"error": "invalid or expired session",
				},
			)
			return
		}

		c.Set(UserIDKey, user.UserID)
		c.Set(OrganizationIDKey, user.OrganizationID)
		c.Set(UserRoleKey, user.Role)
		c.Set(UserEmailKey, user.Email)

		c.Next()
	}
}