package auth

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// RequireAuth is a middleware that ensures the user is authenticated
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")

		if userID == nil {
			// User is not authenticated
			if c.GetHeader("HX-Request") == "true" {
				// HTMX request: send HX-Redirect header
				c.Header("HX-Redirect", "/login")
				c.AbortWithStatus(http.StatusUnauthorized)
			} else {
				// Normal request: redirect to login
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
			}
			return
		}

		// User is authenticated - set context values for downstream handlers
		c.Set("user_id", userID)
		c.Set("user_email", session.Get("user_email"))
		c.Set("user_name", session.Get("user_name"))

		c.Next()
	}
}
