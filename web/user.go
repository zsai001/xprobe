package web

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	db2 "server/db"
	"server/util"
	"time"
)

func User(c *gin.Context) {
	// 从上下文中获取验证过的用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	// 连接到用户集合
	userCollection := db2.MG.CC("prob", "user")

	// 查找用户
	var user DBUser
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	err = userCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
		return
	}

	// 返回用户数据
	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.UserName,
		"isAdmin":  user.IsAdmin,
		"email":    user.Email,
		"twitter":  user.Twitter,
		"telegram": user.Telegram,
	})
}

func Password(c *gin.Context) {
	util.DebugRequest(c)

	var changePasswordRq struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&changePasswordRq); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	// Get the user ID from the token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	userCollection := db2.MG.CC("prob", "user")

	var user DBUser
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("the user id was", userID)
	objId, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}
	err = userCollection.FindOne(ctx, bson.M{"_id": objId}).Decode(&user)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to find user"})
		fmt.Println("find user with err", err.Error())
		return
	}

	// Verify old password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(changePasswordRq.OldPassword))
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid old password"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(changePasswordRq.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to hash new password"})
		return
	}

	// Update password in database
	update := bson.M{"$set": bson.M{"password": string(hashedPassword)}}
	_, err = userCollection.UpdateOne(ctx, bson.M{"_id": objId}, update)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(200, gin.H{"message": "Password updated successfully"})
}
