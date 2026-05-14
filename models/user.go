package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"`             // Don't expose password in JSON
	Role      string    `json:"role" gorm:"default:'analyst'"` // analyst, representative, credit_manager, admin
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HashPassword hashes the password before saving the user
func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	u.Password = string(bytes)
	return nil
}

// CheckPassword checks if the provided password matches the hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// TableName specifies the table name for User
func (User) TableName() string {
	return "users"
}

// BeforeCreate hook to set default role
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Role == "" {
		u.Role = "analyst"
	}
	return nil
}
