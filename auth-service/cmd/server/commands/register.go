package commands

import (
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterCommand struct {
	Email            string
	Password         string
	Name             string
	Dni              string
	Telephone_Number string
	Birthday         time.Time
}

type RegisterHandler struct {
	DB *gorm.DB
}

func (h *RegisterHandler) Handle(cmd RegisterCommand) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(cmd.Password), 12)
	if err != nil {
		return err
	}

	ph := string(hashed)
	user := models.User{
		Email:           &cmd.Email,
		PasswordHash:    &ph,
		Name:            cmd.Name,
		Dni:             cmd.Dni,
		TelephoneNumber: cmd.Telephone_Number,
		Birthday:        cmd.Birthday,
	}

	return h.DB.Create(&user).Error
}
