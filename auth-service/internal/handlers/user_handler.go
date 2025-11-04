package handlers

import (
    "net/http"

    "github.com/Andres09xZ/latacunga_clean_app/internal/database"
    "github.com/Andres09xZ/latacunga_clean_app/internal/models"
    "github.com/gin-gonic/gin"
)

// ListUsers obtiene una lista de todos los usuarios.
// @Summary List all users
// @Description Retrieve a list of all users in the system. Requires admin role.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.User
// @Failure 500 {object} map[string]string
// @Router /api/v1/admin/users [get]
func ListUsers(c *gin.Context) {
    var users []models.User
    if err := database.DB.Find(&users).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
        return
    }
    c.JSON(http.StatusOK, users)
}