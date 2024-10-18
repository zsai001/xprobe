package web

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/url"
	"server/db"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetAddSetting(c *gin.Context) AddSetting {
	hostname := c.Request.Host

	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	baseURL := url.URL{
		Scheme: scheme,
		Host:   hostname,
	}

	cc := db.MG.CC("prob", "node")
	token, err := createOrGetUnboundToken(cc.Collection)
	if err != nil {
		return AddSetting{}
	}
	windowsCmd := fmt.Sprintf("certutil -urlcache -split -f \"%s/install.cmd\" install.cmd && install.cmd %s", baseURL.String(), token)
	linuxCmd := fmt.Sprintf("curl -fsSL %s/install.sh | bash -s %s", baseURL.String(), token)
	macOSCmd := fmt.Sprintf("curl -fsSL %s/install.sh | bash -s %s", baseURL.String(), token)

	return AddSetting{
		Windows: windowsCmd,
		Linux:   linuxCmd,
		MacOS:   macOSCmd,
	}
}

func bindToken(cc *mongo.Collection, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := cc.UpdateOne(
		ctx,
		bson.M{"token": token, "bound": false},
		bson.M{"$set": bson.M{"bound": true}},
	)
	return err
}

func createOrGetUnboundToken(cc *mongo.Collection) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 查找未绑定的 token
	var node Node
	err := cc.FindOne(ctx, bson.M{"bound": false}).Decode(&node)
	if err == nil {
		return node.Token, nil
	}

	// 如果没有找到未绑定的 token，创建一个新的
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	newNode := Node{
		Token:     token,
		CreatedAt: time.Now(),
		Bound:     false,
	}

	_, err = cc.InsertOne(ctx, newNode)
	if err != nil {
		return "", err
	}

	return token, nil
}

func generateToken() (string, error) {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
