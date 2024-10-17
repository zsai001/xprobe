package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"server/client"
	"server/db"
	"server/util"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"server/web"
	"time"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017" // 默认值
	}
	err := db.Init(mongoURI)
	if err != nil {
		panic(err)
	}
	util.Init()

	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	// 构建静态文件目录路径
	staticDir := filepath.Join(currentDir, "html")

	// 创建 Gin 引擎
	r := gin.Default()

	// 添加 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/api/status", web.Status)
	r.GET("/api/user", util.Auth(), web.User)
	r.POST("/api/login", web.Login)
	r.GET("/api/logout", util.Auth(), web.Logout)
	r.POST("/api/user/password", util.Auth(), web.Password)
	r.GET("/api/setting", util.Auth(), web.SettingGet)
	r.POST("/api/setting", util.Auth(), web.SettingSet)
	r.POST("/api/node/delete", util.Auth(), web.DeleteNode)
	r.GET("/install.sh", web.InstallSh)
	r.GET("/install.ps1", web.InstallPs)
	r.GET("/install.cmd", web.InstallCmd)

	r.POST("/api/report/dynamic", client.HandleDynamicReport)
	r.POST("/api/report/static", client.HandleStaticReport)

	// 设置静态文件服务
	r.NoRoute(gin.WrapH(http.FileServer(http.Dir(staticDir))))

	// 启动服务器
	port := 8080
	fmt.Printf("Starting server on :%d, serving files from %s\n", port, staticDir)
	err = r.Run(fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
