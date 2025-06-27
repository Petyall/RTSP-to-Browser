package database

import (
	"fmt"
	"log"
	"path/filepath"

	"rtsp_streamer/config"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	gormpg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func DBConnection() (*gorm.DB, error) {
	err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Ошибка загрузки файла .env: %v", err)
	}

	dbHost := config.GetEnv("DB_HOST")
	dbPort := config.GetEnv("DB_PORT")
	dbUser := config.GetEnv("DB_USER")
	dbPassword := config.GetEnv("DB_PASS")
	dbName := config.GetEnv("DB_NAME")
	dbSSLMode := config.GetEnv("DB_SSLMODE")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode,
	)

	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения sql.DB: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %v", err)
	}

	log.Println("Успешное подключение к базе данных")

	err = runMigrations(dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения миграций: %v", err)
	}

	return db, nil
}

func runMigrations(dsn string) error {
	absPath, err := filepath.Abs("./database/migrations")
	if err != nil {
		return fmt.Errorf("ошибка получения пути к миграциям: %v", err)
	}
	migrationPath := "file://" + absPath

	db, err := gorm.Open(gormpg.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("ошибка подключения для миграций: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("ошибка получения sql.DB для миграций: %v", err)
	}
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("ошибка создания драйвера миграций: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("ошибка инициализации миграций: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("ошибка выполнения миграций: %v", err)
	}

	log.Println("Миграции успешно выполнены")
	return nil
}
