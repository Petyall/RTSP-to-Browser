package database

import (
	"fmt"

	"gorm.io/gorm"
)

func GetCamerasID(db *gorm.DB) ([]Camera, error) {
	var cameras []Camera
	if err := db.Select("id, name").Find(&cameras).Error; err != nil {
		return nil, fmt.Errorf("ошибка при получении списка камер: %v", err)
	}
	return cameras, nil
}

func GetCameraByID(cameraID int, db *gorm.DB) (Camera, error) {
	var camera Camera
	if err := db.Where("id = ?", cameraID).Select("rtsp_url, name").First(&camera).Error; err != nil {
		return Camera{}, fmt.Errorf("ошибка при получении камеры: %v", err)
	}
	return camera, nil
}
