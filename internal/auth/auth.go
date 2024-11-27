package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func CheckPasswordHash(password, hash string) error {
	err :=  bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return err
	}
	return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := &jwt.RegisteredClaims{
		Issuer: "chirpy", 
		Subject: userID.String(),
		IssuedAt: jwt.NewNumericDate(time.Now().UTC()), 
		ExpiresAt:  jwt.NewNumericDate(time.Now().Add(expiresIn).UTC()), 
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return tok, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
		}
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	if !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	fmt.Printf("Got auth header: %s", auth)

	if auth == "" {
		return "", errors.New("no Bearer")
	}
	split := strings.Split(auth, "Bearer ")
	if len(split) == 2 {
		return split[1], nil
	}

	return "", errors.New("invalid bearer") 
}