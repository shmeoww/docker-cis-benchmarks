package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/checks"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
)

func main() {
	// Создаём Docker-клиент один раз при старте.
	// Все обработчики используют его через замыкание.
	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("не удалось подключиться к Docker: %v", err)
	}
	defer cli.Close()

	r := gin.Default()

	// GET /health — проверка работоспособности сервиса
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": checks.Version})
	})

	// POST /scan/image — сканировать один образ
	// Тело: {"image": "nginx:1.27"}
	r.POST("/scan/image", func(c *gin.Context) {
		var req struct {
			Image string `json:"image" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "укажите поле image"})
			return
		}
		ctx := c.Request.Context()
		report, err := checks.ScanImage(ctx, cli, req.Image)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.PureJSON(http.StatusOK, report)
	})

	// POST /scan/container — сканировать один контейнер
	// Тело: {"container": "имя-или-id"}
	r.POST("/scan/container", func(c *gin.Context) {
		var req struct {
			Container string `json:"container" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "укажите поле container"})
			return
		}
		ctx := c.Request.Context()
		report, err := checks.ScanContainer(ctx, cli, req.Container)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.PureJSON(http.StatusOK, report)
	})

	// POST /scan/all — сканировать все образы и контейнеры
	r.POST("/scan/all", func(c *gin.Context) {
		ctx := c.Request.Context()
		reports, err := checks.ScanAll(ctx, cli)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.PureJSON(http.StatusOK, gin.H{"scans": reports, "total": len(reports)})
	})

	log.Println("Сканер запущен на :8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("ошибка запуска сервера: %v", err)
	}
}