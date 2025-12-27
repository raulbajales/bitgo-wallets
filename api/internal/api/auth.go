package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID        uuid.UUID `json:"id"`
		Email     string    `json:"email"`
		FirstName *string   `json:"first_name"`
		LastName  *string   `json:"last_name"`
		Role      string    `json:"role"`
	} `json:"user"`
}

func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// For demo purposes - hardcoded admin user
	if req.Email == s.config.AdminEmail && req.Password == s.config.AdminPassword {
		// Generate a simple token (in production, use JWT)
		token := "demo_token_" + uuid.New().String()

		response := LoginResponse{
			Token: token,
		}
		response.User.ID = uuid.New()
		response.User.Email = req.Email
		firstName := "Admin"
		lastName := "User"
		response.User.FirstName = &firstName
		response.User.LastName = &lastName
		response.User.Role = "admin"

		c.JSON(http.StatusOK, response)
		return
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// For demo purposes - any token starting with "demo_token_" is valid
		if !strings.HasPrefix(token, "demo_token_") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user context (hardcoded for demo)
		c.Set("user_id", uuid.New().String())
		c.Set("user_email", s.config.AdminEmail)
		c.Set("user_role", "admin")

		c.Next()
	})
}

func (s *Server) getCurrentUserID(c *gin.Context) uuid.UUID {
	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))
	return userID
}
