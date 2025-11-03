package queries

import (
	"github.com/Andres09xZ/latacunga_clean_app/internal/models"
	"gorm.io/gorm"
)


type GetByEmailQuery struct {
	Email string
}

type GetByEmailHandler struct {
	DB *gorm.DB
}

func (h *GetByEmailHandler) Handle(q GetByEmailQuery) (*models.User, error) {
	var user models.User

	//Busca por email y solo usuarios activos 
	err := h.DB.Where("email = ? AND is_active = ?", q.Email, true).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}