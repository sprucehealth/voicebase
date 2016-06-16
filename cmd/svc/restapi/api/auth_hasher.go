package api

import (
	"golang.org/x/crypto/bcrypt"
)

const defaultBcryptCost = 10

type PasswordHasher interface {
	GenerateFromPassword(password []byte) ([]byte, error)
	CompareHashAndPassword(hashedPassword, password []byte) error
}

type bcryptHasher struct {
	cost int
}

func NewBcryptHasher(cost int) PasswordHasher {
	if cost < defaultBcryptCost {
		cost = defaultBcryptCost
	}
	return &bcryptHasher{
		cost: cost,
	}
}

func (h *bcryptHasher) GenerateFromPassword(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, h.cost)
}

func (h *bcryptHasher) CompareHashAndPassword(hashedPassword, password []byte) error {
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}
