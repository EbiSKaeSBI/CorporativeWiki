package service

import (
	"time"

	"wiki/internal/models"

	"github.com/golang-jwt/jwt/v5"
)


func (s *Service) GenerateAccessToken(user *models.User) (string, error) {
	now := time.Now()

	exp := now.Add(s.conf.JwtAccessTTL)

	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"role":  user.Role,
		"iat":   jwt.NewNumericDate(now),
		"exp":   jwt.NewNumericDate(exp),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenStr, err := token.SignedString([]byte(s.conf.JwtSecret))
	if err != nil {
		return "", err
	}
	
	return tokenStr, nil

}
