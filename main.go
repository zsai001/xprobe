package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"server/client"
	"server/db"
	"server/util"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"server/web"
	"time"
)

type nextJsFileSystem struct {
	fs http.FileSystem
}

func (nfs nextJsFileSystem) Open(name string) (http.File, error) {
	// URL 解码路径
	decodedPath, err := url.QueryUnescape(name)
	if err != nil {
		return nil, err
	}

	// 移除查询参数
	path := decodedPath
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// 标准化路径，移除开头的 /
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		path = "index.html"
	}

	// 尝试直接打开文件
	if f, err := nfs.fs.Open(path); err == nil {
		return f, nil
	}

	// 如果没有扩展名，尝试以下顺序：
	// 1. 路径.html
	// 2. 路径/index.html
	if !strings.Contains(path, ".") {
		// 尝试 .html 后缀
		if f, err := nfs.fs.Open(path + ".html"); err == nil {
			return f, nil
		}

		// 尝试目录下的 index.html
		if f, err := nfs.fs.Open(filepath.Join(path, "index.html")); err == nil {
			return f, nil
		}
	}

	return nfs.fs.Open(path)
}

// 创建一个日志中间件跟踪访问路径
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}
		log.Printf("访问路径: %s, 状态码: %d\n", path, c.Writer.Status())
	}
}

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
	r.GET("/api/probes", util.Auth(), web.ProbeList)
	r.POST("/api/probes", util.Auth(), web.ProbeAdd)
	r.DELETE("/api/probes/:id", util.Auth(), web.ProbeDelete)
	r.GET("/api/servers/:id", web.GetProbeLatency)

	r.POST("/api/report/dynamic", client.HandleDynamicReport)
	r.POST("/api/report/static", client.HandleStaticReport)

	// 设置静态文件服务
	fs := nextJsFileSystem{http.Dir(staticDir)}
	r.NoRoute(gin.WrapH(http.FileServer(fs)))

	//r.NoRoute(gin.WrapH(http.FileServer(http.Dir(staticDir))))

	// 启动服务器
	port := 8080
	fmt.Printf("Starting server on :%d, serving files from %s\n", port, staticDir)
	err = r.Run(fmt.Sprintf("192.168.31.99:%d", port))
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
