package util

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	db2 "server/db"
)

var jwtSecret = []byte("mFYqxttBhKfNvy1vQ1gZ1+bpB8yJMWoK8o5qQr9JGaQ=")

const (
	TokenExpiration  = 24 * time.Hour
	RenewalThreshold = 12 * time.Hour
	MaxRenewalPeriod = 7 * 24 * time.Hour
)

type Claims struct {
	UserID    string    `json:"user_id"`
	IssuedAt  time.Time `json:"issued_at"`
	RenewedAt time.Time `json:"renewed_at"`
	jwt.StandardClaims
}

type TokenRecord struct {
	Token     string    `bson:"token"`
	UserID    string    `bson:"user_id"`
	ExpiresAt time.Time `bson:"expires_at"`
	RenewedAt time.Time `bson:"renewed_at"`
	IsInvalid bool      `bson:"is_invalid"`
}

var tokenCollection *mongo.Collection

func Init() {
	tokenCollection = db2.MG.CC("prob", "tokens").Collection

	_, err := tokenCollection.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
	if err != nil {
		fmt.Printf("Error creating indexes: %v\n", err)
	}
}

func GenerateToken(userID string) (string, error) {
	now := time.Now()
	expirationTime := now.Add(TokenExpiration)
	claims := &Claims{
		UserID:    userID,
		IssuedAt:  now,
		RenewedAt: now,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  now.Unix(),
			Issuer:    "your-application-name",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	_, err = tokenCollection.InsertOne(context.Background(), TokenRecord{
		Token:     tokenString,
		UserID:    userID,
		ExpiresAt: expirationTime,
		RenewedAt: now,
		IsInvalid: false,
	})
	if err != nil {
		return "", fmt.Errorf("error storing token: %v", err)
	}

	return tokenString, nil
}

func ValidateAndRenewToken(tokenString string) (string, *Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return "", nil, fmt.Errorf("invalid token")
	}

	var record TokenRecord
	err = tokenCollection.FindOne(context.Background(), bson.M{"token": tokenString}).Decode(&record)
	if err != nil {
		return "", nil, fmt.Errorf("token not found in database")
	}

	if record.IsInvalid {
		return "", nil, fmt.Errorf("token has been invalidated")
	}

	now := time.Now()
	if now.Sub(claims.RenewedAt) > MaxRenewalPeriod {
		return "", nil, fmt.Errorf("token has exceeded maximum renewal period")
	}

	if now.Add(RenewalThreshold).Before(time.Unix(claims.ExpiresAt, 0)) {
		return tokenString, claims, nil // Token is still valid and doesn't need renewal
	}

	// Renew the token
	newExpirationTime := now.Add(TokenExpiration)
	claims.ExpiresAt = newExpirationTime.Unix()
	claims.RenewedAt = now

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	newTokenString, err := newToken.SignedString(jwtSecret)
	if err != nil {
		return "", nil, fmt.Errorf("error signing new token: %v", err)
	}

	// Update the database
	_, err = tokenCollection.UpdateOne(
		context.Background(),
		bson.M{"token": tokenString},
		bson.M{
			"$set": bson.M{
				"token":      newTokenString,
				"expires_at": newExpirationTime,
				"renewed_at": now,
			},
		},
	)
	if err != nil {
		return "", nil, fmt.Errorf("error updating token in database: %v", err)
	}

	return newTokenString, claims, nil
}

func InvalidateToken(tokenString string) error {
	_, err := tokenCollection.UpdateOne(
		context.Background(),
		bson.M{"token": tokenString},
		bson.M{"$set": bson.M{"is_invalid": true}},
	)
	return err
}

func GetUserIDFromToken(tokenString string) (string, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
