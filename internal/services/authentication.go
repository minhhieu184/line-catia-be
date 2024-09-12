package services

import (
	"encoding/json"
	"errors"
	"millionaire/internal/models"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Authentication struct {
	secret string
}

func NewAuthentication(secret string) (*Authentication, error) {
	return &Authentication{secret}, nil
}

func (authentication *Authentication) CreateToken(user *models.UserFromAuth) (string, error) {
	claims := jwt.MapClaims{
		"id":            user.ID,
		"username":      user.Username,
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"is_bot":        user.IsBot,
		"is_premium":    user.IsPremium,
		"language_code": user.LanguageCode,
		"photo_url":     user.PhotoURL,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(authentication.secret))
}

func (authentication *Authentication) Validate(token string) (*models.UserFromAuth, error) {
	println("token xxx", token)

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	}
	jwtToken, err := jwt.ParseWithClaims(token, &CustomClaims{}, keyFunc)
	if err != nil {
		return nil, err
	}

	claims, ok := jwtToken.Claims.(*CustomClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	b, _ := json.MarshalIndent(claims, "", "    ")
	println("claims", string(b))

	return &models.UserFromAuth{
		ID:           claims.ID,
		Username:     claims.Username,
		// FirstName:    data.User.FirstName,
		// LastName:     data.User.LastName,
		// IsBot:        data.User.IsBot,
		// IsPremium:    data.User.IsPremium,
		// LanguageCode: data.User.LanguageCode,
		// PhotoURL:     data.User.PhotoURL,
	}, nil
}
