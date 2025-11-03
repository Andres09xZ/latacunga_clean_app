package repository

import (
	"errors"

	"github.com/Andres09xZ/latacunga_clean_app/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CreateUser crea un usuario en la base de datos. Recibe la estructura User
// y la contraseña en claro; la contraseña se hashea antes de guardar.
func CreateUser(u *models.User, plainPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hs := string(hash)
	u.PasswordHash = &hs
	return database.DB.Create(u).Error
}

// GetUserByID devuelve el usuario por su ID o nil si no existe.
func GetUserByID(id uint) (*models.User, error) {
	var u models.User
	if err := database.DB.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// GetUserByEmail busca un usuario por email. Devuelve nil,nil si no existe.
func GetUserByEmail(email string) (*models.User, error) {
	var u models.User
	if err := database.DB.Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// UpdateUser actualiza los campos del usuario (suponer que la estructura ya tiene ID).
// No toca la contraseña; para cambiarla usar ChangePassword.
func UpdateUser(u *models.User) error {
	return database.DB.Save(u).Error
}

// DeleteUser elimina el usuario por ID (soft delete si GORM está configurado).
func DeleteUser(id uint) error {
	return database.DB.Delete(&models.User{}, id).Error
}

// Authenticate verifica email + password. Devuelve el usuario si coincide, nil,nil si
// credenciales inválidas, o error en caso de fallo de DB.
func Authenticate(email, plainPassword string) (*models.User, error) {
	u, err := GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		// usuario no existe
		return nil, nil
	}
	if u.PasswordHash == nil {
		return nil, nil
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(plainPassword)); err != nil {
		// contraseña no coincide
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

// ChangePassword cambia la contraseña de un usuario (recibe el ID y la nueva contraseña).
func ChangePassword(id uint, newPlain string) error {
	u, err := GetUserByID(id)
	if err != nil {
		return err
	}
	if u == nil {
		return gorm.ErrRecordNotFound
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPlain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hs := string(hash)
	u.PasswordHash = &hs
	return UpdateUser(u)
}
