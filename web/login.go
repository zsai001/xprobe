package web

import (
	"context"
	"net/http"
	db2 "server/db"
	"server/util"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type LoginRq struct {
	Password string `json:"password" binding:"required"`
	UserName string `json:"username" binding:"required"`
}

type LoginRs struct {
	Token string `json:"token"`
	User  DBUser `json:"user"`
}

type DBUser struct {
	ID       string `bson:"_id,omitempty" json:"id"`
	UserName string `bson:"username" json:"username"`
	IsAdmin  bool   `bson:"isAdmin" json:"isAdmin"`
	Email    string `bson:"email,omitempty" json:"email,omitempty"`
	Twitter  string `bson:"twitter,omitempty" json:"twitter,omitempty"`
	Telegram string `bson:"telegram,omitempty" json:"telegram,omitempty"`
	Password string `bson:"password" json:"-"` // 不在 JSON 响应中返回密码
}

func InitAdminUser(userCollection *mongo.Collection) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingAdmin DBUser
	err := userCollection.FindOne(ctx, bson.M{"username": "admin"}).Decode(&existingAdmin)
	if err == nil {
		return nil
	}

	if err != mongo.ErrNoDocuments {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	adminUser := DBUser{
		UserName: "admin",
		Password: string(hashedPassword),
		IsAdmin:  true,
		Email:    "admin@example.com", // 可以根据需要修改
		Twitter:  "@zsai010",
		Telegram: "@cyberstan",
	}

	_, err = userCollection.InsertOne(ctx, adminUser)
	return err
}

func Login(c *gin.Context) {
	util.DebugRequest(c)

	var loginRq LoginRq
	if err := c.ShouldBindJSON(&loginRq); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body"})
		return
	}

	userCollection := db2.MG.CC("prob", "user")

	if err := InitAdminUser(userCollection.Collection); err != nil {
		c.JSON(500, gin.H{"error": "Failed to initialize admin user"})
		return
	}

	var user DBUser
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := userCollection.FindOne(ctx, bson.M{"username": loginRq.UserName}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(401, gin.H{"error": "Invalid username or password"})
		} else {
			c.JSON(500, gin.H{"error": "Internal server error"})
		}
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRq.Password))
	if err != nil {
		c.JSON(401, gin.H{"error": "Invalid username or password"})
		return
	}

	token, err := util.GenerateToken(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate token"})
		return
	}

	loginRs := LoginRs{
		Token: token,
		User: DBUser{
			ID:       user.ID,
			UserName: user.UserName,
			IsAdmin:  user.IsAdmin,
			Email:    user.Email,
			Twitter:  user.Twitter,
			Telegram: user.Telegram,
		},
	}

	c.JSON(200, loginRs)
}

func Logout(c *gin.Context) {
	// 从上下文中获取用户ID（由Auth中间件设置）
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	// 获取当前的token
	token := c.GetHeader("Authorization")
	if token == "" || len(token) <= 7 || token[:7] != "Bearer " {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token format"})
		return
	}
	token = token[7:] // 移除 "Bearer " 前缀

	// 将token加入黑名单或使其失效
	err := util.InvalidateToken(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
