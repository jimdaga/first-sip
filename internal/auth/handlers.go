package auth

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jimdaga/first-sip/internal/models"
	"github.com/markbates/goth/gothic"
	"gorm.io/gorm"
)

// HandleLogin initiates the Google OAuth flow
func HandleLogin(c *gin.Context) {
	// Gothic requires the "provider" query parameter
	q := c.Request.URL.Query()
	q.Add("provider", "google")
	c.Request.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

// HandleCallback completes the OAuth flow, upserts the user, and stores info in session
func HandleCallback(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Gothic requires the "provider" query parameter
		q := c.Request.URL.Query()
		q.Add("provider", "google")
		c.Request.URL.RawQuery = q.Encode()

		gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
		if err != nil {
			log.Printf("Auth error: %v", err)
			c.Redirect(http.StatusFound, "/login?error=auth_failed")
			return
		}

		// Upsert user record in database
		if db != nil {
			now := time.Now()
			var user models.User
			result := db.Where("email = ?", gothUser.Email).First(&user)
			if result.Error == gorm.ErrRecordNotFound {
				user = models.User{
					Email:       gothUser.Email,
					Name:        gothUser.Name,
					LastLoginAt: &now,
				}
				db.Create(&user)
			} else if result.Error == nil {
				db.Model(&user).Updates(map[string]interface{}{
					"name":          gothUser.Name,
					"last_login_at": now,
				})
			}
		}

		// Store user info in session
		session := sessions.Default(c)
		session.Set("user_id", gothUser.UserID)
		session.Set("user_email", gothUser.Email)
		session.Set("user_name", gothUser.Name)
		session.Set("user_avatar", gothUser.AvatarURL)

		if err := session.Save(); err != nil {
			log.Printf("Session save error: %v", err)
			c.Redirect(http.StatusFound, "/login?error=session_failed")
			return
		}

		log.Printf("User authenticated: %s (%s)", gothUser.Name, gothUser.Email)
		c.Redirect(http.StatusFound, "/dashboard")
	}
}

// HandleLogout clears the session and redirects to login
func HandleLogout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()

	if err := session.Save(); err != nil {
		log.Printf("Session clear error: %v", err)
	}

	c.Redirect(http.StatusFound, "/login")
}
