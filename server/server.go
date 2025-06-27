package server

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"rtsp_streamer/database"
	"rtsp_streamer/storage"

	"github.com/gin-gonic/gin"
)

// SetupServer настраивает и возвращает HTTP-сервер с маршрутами и middleware
func SetupServer(cameras []database.Camera) *http.Server {
	// Инициализация Gin
	r := gin.Default()

	// Настройка middleware для заголовков кэширования
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Writer.Header().Set("Surrogate-Control", "no-store")
		c.Next()
	})

	// Настройка шаблонов из папки templates
	tmpl := template.Must(template.ParseFiles(filepath.Join("templates", "index.html")))

	// Настройка маршрутов
	r.StaticFS("/hls", gin.Dir(storage.HLSPath, false))
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		// Данные для шаблона
		data := struct {
			Title string
		}{
			Title: "VisionSecure",
		}
		if err := tmpl.ExecuteTemplate(c.Writer, "index.html", data); err != nil {
			c.String(http.StatusInternalServerError, "Ошибка рендеринга шаблона: %v", err)
			return
		}
	})

	r.GET("/api/cameras", func(c *gin.Context) {
		var camerasList []database.CameraAPI
		for _, camera := range cameras {
			camerasList = append(camerasList, database.CameraAPI{
				ID:   camera.ID,
				Name: camera.Name,
				URL:  fmt.Sprintf("/hls/camera_%d/stream.m3u8", camera.ID),
			})
		}
		c.JSON(200, camerasList)
	})

	// Создание HTTP-сервера
	return &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r,
	}
}
