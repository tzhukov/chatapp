package crypto

import (
	"golang.org/x/crypto/bcrypt"
)

// Hash returns a bcrypt hash of the given password using DefaultCost.
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare returns true if the provided password matches the given bcrypt hash.
func Compare(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
