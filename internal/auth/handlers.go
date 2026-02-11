package auth

import (
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

// HandleLogin initiates the Google OAuth flow
func HandleLogin(c *gin.Context) {
	// Gothic requires the "provider" query parameter
	q := c.Request.URL.Query()
	q.Add("provider", "google")
	c.Request.URL.RawQuery = q.Encode()

	gothic.BeginAuthHandler(c.Writer, c.Request)
}

// HandleCallback completes the OAuth flow and stores user info in session
func HandleCallback(c *gin.Context) {
	// Gothic requires the "provider" query parameter
	q := c.Request.URL.Query()
	q.Add("provider", "google")
	c.Request.URL.RawQuery = q.Encode()

	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		log.Printf("Auth error: %v", err)
		c.Redirect(http.StatusFound, "/login?error=auth_failed")
		return
	}

	// Store user info in session
	session := sessions.Default(c)
	session.Set("user_id", user.UserID)
	session.Set("user_email", user.Email)
	session.Set("user_name", user.Name)
	session.Set("user_avatar", user.AvatarURL)

	if err := session.Save(); err != nil {
		log.Printf("Session save error: %v", err)
		c.Redirect(http.StatusFound, "/login?error=session_failed")
		return
	}

	log.Printf("User authenticated: %s (%s)", user.Name, user.Email)
	c.Redirect(http.StatusFound, "/dashboard")
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
