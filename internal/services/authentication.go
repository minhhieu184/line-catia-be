package services

import (
	"millionaire/internal/models"

	"github.com/golang-jwt/jwt"
)

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
	// err := initdata.Validate(dataStr, bot.token, 0)
	// if err != nil {
	// 	return nil, err
	// }

	println("token xxx", token)

	// claims, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
	// 	return []byte(authentication.secret), nil
	// })
	// if err != nil {
	// 	return nil, err
	// }

	return &models.UserFromAuth{
		// ID:           data.User.ID,
		// Username:     data.User.Username,
		// FirstName:    data.User.FirstName,
		// LastName:     data.User.LastName,
		// IsBot:        data.User.IsBot,
		// IsPremium:    data.User.IsPremium,
		// LanguageCode: data.User.LanguageCode,
		// PhotoURL:     data.User.PhotoURL,
	}, nil
}
