package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/iiwish/data-anonymization/internal/config"
	"github.com/iiwish/data-anonymization/internal/handler"
	"github.com/iiwish/data-anonymization/internal/logger"
	"github.com/iiwish/data-anonymization/internal/middleware"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.json", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Printf("配置加载成功，已配置 %d 个系统", len(cfg.Systems))

	// 初始化日志
	if err := logger.Init(cfg.Server.LogFile); err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}
	defer logger.Close()

	logger.Info("Server", "服务启动中...")

	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	router := gin.Default()

	// 添加CORS中间件（如果需要）
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 创建处理器
	anonymizationHandler := handler.NewAnonymizationHandler()
	decryptionHandler := handler.NewDecryptionHandler()

	// 配置路由
	v1 := router.Group("/v1")
	{
		// 匿名化接口
		v1.POST("/anonymize", middleware.HMACAuth(cfg.Server.TimestampWindowSeconds), anonymizationHandler.Handle)

		// 解密接口
		v1.POST("/decrypt", middleware.HMACAuth(cfg.Server.TimestampWindowSeconds), decryptionHandler.Handle)
	}

	// 健康检查接口（不需要鉴权）
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("服务器启动在端口 %d", cfg.Server.Port)
	logger.Info("Server", fmt.Sprintf("服务器启动在端口 %d", cfg.Server.Port))

	// 优雅关闭
	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server", "服务器正在关闭...")
	log.Println("服务器正在关闭...")
}
