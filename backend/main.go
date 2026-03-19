package main

import (
	"time"

	"github.com/gin-contrib/cors" // 跨域插件
	"github.com/gin-gonic/gin"    // 修正了拼写
)

func main() {
	r := gin.Default()

	// 配置跨域中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour, // 这里必须是英文冒号
	}))

	// 测试接口
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "握手成功 Flashwiki 后端就绪",
		})
	})

	r.Run(":8080")
}
