package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"rtsp_streamer/database"
	"rtsp_streamer/ffmpeg"
	"rtsp_streamer/server"
	"rtsp_streamer/storage"
)

func main() {
	storage.CheckHLSFolder()

	// Подключение к базе данных
	db, err := database.DBConnection()
	if err != nil {
		log.Fatalf("Ошибка при подключении к базе данных: %v", err)
	}

	// Получение sql.DB для закрытия соединения
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Ошибка при получении sql.DB: %v", err)
	}

	// Автомиграция
	if err := db.AutoMigrate(&database.Camera{}); err != nil {
		log.Fatalf("Ошибка при автомиграции: %v", err)
	}

	// Получение списка камер
	cameras, err := database.GetCamerasID(db)
	if err != nil {
		log.Fatalf("Ошибка при получении камер: %v", err)
	}

	fmt.Println("Список камер:")
	for _, camera := range cameras {
		fmt.Printf("ID: %d, Name: %s\n", camera.ID, camera.Name)
	}

	// Инициализация и запуск сервера
	server := server.SetupServer(cameras)

	// Создание контекста для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Запуск FFmpeg-процессов с контекстом
	for _, camera := range cameras {
		ffmpeg.StartFFmpeg(camera.ID, db, ctx)
	}

	// Запуск сервера в отдельной горутине
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидание сигнала завершения
	<-ctx.Done()
	log.Println("Получен сигнал завершения, начинаем graceful shutdown...")

	// Создание контекста для завершения с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Остановка всех FFmpeg-процессов
	log.Println("Остановка FFmpeg-процессов...")
	ffmpeg.StopAllFFmpeg()

	// Завершение работы HTTP-сервера
	log.Println("Остановка HTTP-сервера...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Ошибка при завершении работы сервера: %v", err)
	}

	// Закрытие соединения с базой данных
	log.Println("Закрытие соединения с базой данных...")
	if err := sqlDB.Close(); err != nil {
		log.Printf("Ошибка при закрытии соединения с базой данных: %v", err)
	}

	log.Println("Приложение успешно остановлено")
}
