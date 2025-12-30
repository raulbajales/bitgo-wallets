package api

import (
	"net/http"

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
		// DISABLED: Authentication completely disabled for testing
		// Just pass through without any checks
		c.Next()
	})
}

func (s *Server) getCurrentUserID(c *gin.Context) uuid.UUID {
	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))
	return userID
}
