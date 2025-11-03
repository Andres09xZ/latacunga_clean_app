package commands

import (
	"errors"

	"github.com/Andres09xZ/latacunga_clean_app/internal/auth"
	"github.com/Andres09xZ/latacunga_clean_app/internal/cqrs/queries"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type LoginCommand struct {
	Email    string
	Password string
}

type LoginHandler struct {
	QueryHandler *queries.GetByEmailHandler
	DB           *gorm.DB
}

func (h *LoginHandler) Handle(cmd LoginCommand) (string, string, error) {
	user, err := h.QueryHandler.Handle(queries.GetByEmailQuery{Email: cmd.Email})
	if err != nil {
		return "", "", err
	}
	if user == nil {
		return "", "", errors.New("invalid credentials")
	}

	// Ensure password hash exists and verify
	if user.PasswordHash == nil {
		return "", "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(cmd.Password)); err != nil {
		return "", "", errors.New("invalid credentials")
	}

	// Prepare email string for tokens (nullable)
	emailStr := ""
	if user.Email != nil {
		emailStr = *user.Email
	}

	// Generate tokens
	access, refresh, err := auth.GenerateTokens(user.IDUser, emailStr, user.Role)
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}
