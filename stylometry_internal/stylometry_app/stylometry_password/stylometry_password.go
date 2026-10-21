package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// Hash превращает обычный пароль в bcrypt-хеш для хранения в БД.
func Hash(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("пароль не может быть пустым")
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(raw),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

// Check сравнивает введённый пароль с bcrypt-хешем из базы данных.
func Check(raw string, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}
