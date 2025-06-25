package ffmpeg

import (
	"database/sql"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"rtsp_streamer/database"
	"rtsp_streamer/utils"
)

// startFFmpeg запускает поток в отдельной горутине.
func StartFFmpeg(cameraID int, db *sql.DB) {
	go startFFmpegProcess(cameraID, db)
}

// startFFmpegProcess запускает процесс ffmpeg и следит за его состоянием.
func startFFmpegProcess(cameraID int, db *sql.DB) {
	for {
		utils.CheckCameraFolder(cameraID)

		codec, bitRate, width, height, frameRate, rtspURL, err := getStreamInfo(cameraID, db)
		if err != nil {
			log.Printf("Ошибка получения параметров для камеры %d: %v", cameraID, err)
			time.Sleep(10 * time.Second) // Повторная попытка через 10 секунд
			continue
		}

		log.Printf("Камера %d | Кодек: %s | Битрейт: %d | Разрешение: %dx%d | FPS: %.2f",
			cameraID, codec, bitRate, width, height, frameRate)

		args := []string{
			"-rtsp_transport", "tcp",
			"-i", rtspURL,
			"-rtbufsize", "20M",
			"-probesize", "2000000",
			"-analyzeduration", "500000",
			"-g", "50",
			"-force_key_frames", "expr:gte(t,n_forced*2)",
			"-hls_time", "2",
			"-hls_list_size", "5",
			"-hls_flags", "append_list+delete_segments",
			"-an",
		}

		if codec == "hevc" || codec == "h265" {
			args = append(args, "-c:v", "h264_nvenc", "-preset", "fast", "-b:v", "2M")
		} else {
			args = append(args, "-c:v", "copy")
		}

		if bitRate > 4000000 {
			args = append(args, "-rtbufsize", "50M")
		}

		if frameRate > 30 {
			args = append(args, "-r", "30")
		}

		args = append(args, utils.HLSPath+"/camera_"+strconv.Itoa(cameraID)+"/stream.m3u8")

		cmd := exec.Command("ffmpeg", args...)

		// Захват stderr для мониторинга ошибок
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("Ошибка при получении stderr для камеры %d: %v", cameraID, err)
			time.Sleep(10 * time.Second)
			continue
		}

		if err := cmd.Start(); err != nil {
			log.Printf("Ошибка при запуске ffmpeg для камеры %d: %v", cameraID, err)
			time.Sleep(10 * time.Second)
			continue
		}

		log.Printf("FFmpeg успешно запущен для камеры %d", cameraID)

		// Следим за stderr в горутине для отслеживания ошибок
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stderr.Read(buf)
				if err != nil {
					log.Printf("Ошибка при чтении stderr для камеры %d: %v", cameraID, err)
					return
				}
				if n > 0 {
					log.Printf("stderr для камеры %d: %s", cameraID, string(buf[:n]))
				}
			}
		}()

		// Блокирующее ожидание завершения процесса
		err = cmd.Wait()
		if err != nil {
			log.Printf("FFmpeg завершился с ошибкой для камеры %d: %v", cameraID, err)
		} else {
			log.Printf("FFmpeg завершился для камеры %d, перезапуск...", cameraID)
		}

		// Ожидание перед перезапуском
		time.Sleep(5 * time.Second)
	}
}

// getStreamInfo получает информацию о потоке через ffprobe.
func getStreamInfo(cameraID int, db *sql.DB) (string, int, int, int, float64, string, error) {
	camera, err := database.GetCameraByID(cameraID, db)
	if err != nil {
		return "", 0, 0, 0, 0, "", err
	}
	rtspURL := camera.RtspURL

	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name,bit_rate,width,height,r_frame_rate",
		"-of", "csv=p=0",
		rtspURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return "", 0, 0, 0, 0, "", err
	}

	fields := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(fields) < 5 {
		return "", 0, 0, 0, 0, "", fmt.Errorf("не удалось получить данные о потоке")
	}

	codec := fields[0]
	bitRate, _ := strconv.Atoi(fields[1])
	width, _ := strconv.Atoi(fields[2])
	height, _ := strconv.Atoi(fields[3])

	frameRateParts := strings.Split(fields[4], "/")
	var frameRate float64 = 0
	if len(frameRateParts) == 2 {
		num, _ := strconv.ParseFloat(frameRateParts[0], 64)
		den, _ := strconv.ParseFloat(frameRateParts[1], 64)
		if den > 0 {
			frameRate = num / den
		}
	}

	return codec, bitRate, width, height, frameRate, rtspURL, nil
}
