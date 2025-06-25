package database

import (
	"database/sql"
	"fmt"
	"log"
	"rtsp_streamer/config"
)

func connect() (*sql.DB, error) {
	dbHost := config.GetEnv("DB_HOST")
	dbPort := config.GetEnv("DB_PORT")
	dbUser := config.GetEnv("DB_USER")
	dbPassword := config.GetEnv("DB_PASS")
	dbName := config.GetEnv("DB_NAME")
	dbSSLMode := config.GetEnv("DB_SSLMODE")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode,
	)

	return sql.Open("postgres", connStr)
}

func DBConnection() (*sql.DB, error) {
	err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Ошибка загрузки файла .env: %v", err)
	}

	db, err := connect()
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %v", err)
	}
	log.Println("Успешное подключение к базе данных")

	return db, nil
}

func GetCamerasID(db *sql.DB) ([]Camera, error) {
	rows, err := db.Query("SELECT id FROM cameras")
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка камер: %v", err)
	}
	defer rows.Close()

	var cameras []Camera
	for rows.Next() {
		var camera Camera
		if err := rows.Scan(&camera.ID); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании данных: %v", err)
		}
		cameras = append(cameras, camera)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при обработке строк: %v", err)
	}

	return cameras, nil
}

func GetCameraByID(cameraID int, db *sql.DB) (Camera, error) {
	query := "SELECT rtsp_url FROM cameras WHERE id = $1"

	row := db.QueryRow(query, cameraID)

	var camera Camera

	err := row.Scan(&camera.RtspURL)
	if err != nil {
		return Camera{}, err
	}

	return camera, nil
}
