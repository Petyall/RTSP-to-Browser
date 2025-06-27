package storage

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
)

const HLSPath = "./static/hls_segments"

func CheckHLSFolder() {
	err := os.RemoveAll(HLSPath)
	if err != nil {
		log.Fatal(err)
	}

	err = os.Mkdir(HLSPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}

func CheckCameraFolder(cameraID int) {
	dirPath := filepath.Join(HLSPath, "camera_"+strconv.Itoa(cameraID))
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
}
