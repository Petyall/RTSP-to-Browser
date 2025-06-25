package main

import (
	"fmt"
	"log"

	"rtsp_streamer/database"
	"rtsp_streamer/ffmpeg"
	"rtsp_streamer/utils"

	"github.com/gin-gonic/gin"

	_ "github.com/lib/pq"
)

func main() {
	utils.CheckHLSFolder()

	db, err := database.DBConnection()
	if err != nil {
		log.Fatalf("Ошибка при подключении к базе данных: %v", err)
	}
	defer db.Close()

	cameras, err := database.GetCamerasID(db)
	if err != nil {
		log.Fatalf("Ошибка при получении камер: %v", err)
	}

	fmt.Println("Список камер:")
	for _, camera := range cameras {
		fmt.Printf("ID: %d\n", camera.ID)
	}

	r := gin.Default()

	for _, camera := range cameras {
		ffmpeg.StartFFmpeg(camera.ID, db)
	}

	r.StaticFS("/hls_files", gin.Dir(utils.HLSPath, false))
	r.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/hls_files" {
			c.Writer.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		}
		c.Next()
	})

	r.Static("/static", "./static")

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Writer.Header().Set("Surrogate-Control", "no-store")
		c.Next()
	})

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.GET("/api/cameras", func(c *gin.Context) {
		var camerasList []database.CameraAPI
		for _, camera := range cameras {
			camerasList = append(camerasList, database.CameraAPI{
				ID:   camera.ID,
				Name: fmt.Sprintf("Camera %d", camera.ID),
				URL:  fmt.Sprintf("/hls_files/camera_%d/stream.m3u8", camera.ID),
			})
		}
		c.JSON(200, camerasList)
	})

	r.LoadHTMLFiles("./templates/index.html")

	log.Fatal(r.Run("0.0.0.0:8080"))
}
