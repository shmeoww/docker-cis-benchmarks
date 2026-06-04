package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/moby/moby/client"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/checks"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
)

// setupRouter создаёт и настраивает Gin-роутер со всеми эндпоинтами
// Вынесено отдельно, чтобы можно было тестировать без запуска сервера
func setupRouter(cli *client.Client) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": checks.Version})
	})

	r.POST("/scan/image", func(c *gin.Context) {
		var req struct {
			Image string `json:"image" binding:"required"`
		}
		// 400 — клиент прислал плохой запрос
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "укажите поле image"})
			return
		}
		// 500 — образ не найден или Docker недоступен
		report, err := checks.ScanImage(context.Background(), cli, req.Image)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.PureJSON(http.StatusOK, report)
	})

	r.POST("/scan/container", func(c *gin.Context) {
		var req struct {
			Container string `json:"container" binding:"required"`
		}
		// 400 — клиент прислал плохой запрос
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "укажите поле container"})
			return
		}
		// 500 — образ не найден или Docker недоступен
		report, err := checks.ScanContainer(context.Background(), cli, req.Container)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.PureJSON(http.StatusOK, report)
	})

	r.POST("/scan/all", func(c *gin.Context) {
		reports, err := checks.ScanAll(context.Background(), cli)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.PureJSON(http.StatusOK, gin.H{"scans": reports, "total": len(reports)})
	})

	return r
}

func main() {
	cli, err := docker.NewClient()
	if err != nil {
		log.Fatalf("не удалось подключиться к Docker: %v", err)
	}
	defer cli.Close()

	r := setupRouter(cli)
	log.Println("Сканер запущен на :8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("ошибка запуска сервера: %v", err)
	}
}