package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	services "example.com/internal/services/db"
)

type AuthHandler struct {
	service services.AuthService
}

func NewAuthHandler(service services.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

type setupRequest struct {
	OrganizationName string `json:"organization_name" binding:"required"`
	Domain           string `json:"domain"`
	Name             string `json:"name" binding:"required"`
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type invitationRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
	Role  string `json:"role" binding:"required"`
}

type acceptInvitationRequest struct {
	Token    string `json:"token" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type updateUserRequest struct {
	Name     string `json:"name" binding:"required"`
	IsActive bool   `json:"is_active"`
}

type updateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

func (h *AuthHandler) Setup(c *gin.Context) {
	var req setupRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := h.service.Setup(
		c.Request.Context(),
		req.OrganizationName,
		req.Domain,
		req.Name,
		req.Email,
		req.Password,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, token, err := h.service.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
	})

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, err := c.Cookie("session")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "logged out",
		})
		return
	}

	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out",
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	token, err := c.Cookie("session")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	user, err := h.service.GetSessionUser(
		c.Request.Context(),
		token,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) CreateInvitation(c *gin.Context) {
	var req invitationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	organizationID, err := getOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid organization id",
		})
		return
	}

	token, err := h.service.CreateInvitation(
		c.Request.Context(),
		organizationID,
		req.Email,
		req.Name,
		req.Role,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
	})
}

func (h *AuthHandler) AcceptInvitation(c *gin.Context) {
	var req acceptInvitationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := h.service.AcceptInvitation(
		c.Request.Context(),
		req.Token,
		req.Name,
		req.Password,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	user, err := h.service.GetUser(
		c.Request.Context(),
		userID,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	var req updateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := h.service.UpdateUser(
		c.Request.Context(),
		userID,
		req.Name,
		req.IsActive,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}

	organizationID, err := getOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid organization id",
		})
		return
	}

	var req updateRoleRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.service.UpdateUserRole(
		c.Request.Context(),
		organizationID,
		userID,
		req.Role,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user role updated",
	})
}

func (h *AuthHandler) ListOrganizationMembers(c *gin.Context) {
	organizationID, err := getOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid organization id",
		})
		return
	}

	users, err := h.service.ListOrganizationMembers(
		c.Request.Context(),
		organizationID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}

func getOrganizationID(c *gin.Context) (uuid.UUID, error) {
	value, exists := c.Get("organization_id")
	if !exists {
		return uuid.Nil, errors.New("organization ID not found")
	}

	switch id := value.(type) {
	case uuid.UUID:
		return id, nil
	case string:
		return uuid.Parse(id)
	default:
		return uuid.Nil, errors.New("invalid organization ID")
	}
}
