package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"rtsp_streamer/database"
	"rtsp_streamer/storage"

	"gorm.io/gorm"
)

type StreamInfo struct {
	Codec     string
	BitRate   int
	Width     int
	Height    int
	FrameRate float64
	RtspURL   string
}

// ffmpegProcesses хранит активные FFmpeg-процессы для каждой камеры.
var (
	ffmpegProcesses = make(map[int]*exec.Cmd)
	processesMu     sync.Mutex
)

// StartFFmpeg запускает поток в отдельной горутине с поддержкой контекста.
func StartFFmpeg(cameraID int, db *gorm.DB, ctx context.Context) {
	go startFFmpegProcess(cameraID, db, ctx)
}

// startFFmpegProcess управляет жизненным циклом процесса FFmpeg.
func startFFmpegProcess(cameraID int, db *gorm.DB, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Получен сигнал завершения для камеры %d, останавливаем FFmpeg", cameraID)
			StopFFmpeg(cameraID)
			return
		default:
			storage.CheckCameraFolder(cameraID)

			streamInfo, err := getStreamInfo(cameraID, db)
			if err != nil {
				log.Printf("Ошибка получения параметров для камеры %d: %v", cameraID, err)
				time.Sleep(10 * time.Second)
				continue
			}

			logStreamInfo(cameraID, streamInfo)

			cmd, err := startFFmpeg(cameraID, streamInfo)
			if err != nil {
				log.Printf("Ошибка при запуске FFmpeg для камеры %d: %v", cameraID, err)
				time.Sleep(10 * time.Second)
				continue
			}

			// Сохраняем процесс
			processesMu.Lock()
			ffmpegProcesses[cameraID] = cmd
			processesMu.Unlock()

			err = cmd.Wait()
			if err != nil {
				log.Printf("FFmpeg завершился с ошибкой для камеры %d: %v", cameraID, err)
			} else {
				log.Printf("FFmpeg завершился для камеры %d, перезапуск...", cameraID)
			}

			// Удаляем процесс из списка после завершения
			processesMu.Lock()
			delete(ffmpegProcesses, cameraID)
			processesMu.Unlock()

			select {
			case <-ctx.Done():
				log.Printf("Получен сигнал завершения для камеры %d, останавливаем FFmpeg", cameraID)
				return
			case <-time.After(5 * time.Second):
				// Продолжаем цикл после паузы
			}
		}
	}
}

// StopFFmpeg останавливает FFmpeg-процесс для указанной камеры.
func StopFFmpeg(cameraID int) {
	processesMu.Lock()
	cmd, exists := ffmpegProcesses[cameraID]
	if exists {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			log.Printf("Ошибка при отправке SIGINT процессу FFmpeg для камеры %d: %v", cameraID, err)
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Ошибка при принудительном завершении FFmpeg для камеры %d: %v", cameraID, err)
			}
		}
		delete(ffmpegProcesses, cameraID)
	}
	processesMu.Unlock()
}

// StopAllFFmpeg останавливает все FFmpeg-процессы.
func StopAllFFmpeg() {
	processesMu.Lock()
	for cameraID, cmd := range ffmpegProcesses {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			log.Printf("Ошибка при отправке SIGINT процессу FFmpeg для камеры %d: %v", cameraID, err)
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Ошибка при принудительном завершении FFmpeg для камеры %d: %v", cameraID, err)
			}
		}
		delete(ffmpegProcesses, cameraID)
	}
	processesMu.Unlock()
}

// getStreamInfo получает информацию о потоке через ffprobe.
func getStreamInfo(cameraID int, db *gorm.DB) (StreamInfo, error) {
	camera, err := database.GetCameraByID(cameraID, db)
	if err != nil {
		return StreamInfo{}, err
	}

	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name,bit_rate,width,height,r_frame_rate",
		"-of", "csv=p=0",
		camera.RtspURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return StreamInfo{}, err
	}

	fields := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(fields) < 5 {
		return StreamInfo{}, fmt.Errorf("не удалось получить данные о потоке")
	}

	codec := fields[0]
	bitRate, _ := strconv.Atoi(fields[1])
	width, _ := strconv.Atoi(fields[2])
	height, _ := strconv.Atoi(fields[3])

	var frameRate float64
	if frameRateParts := strings.Split(fields[4], "/"); len(frameRateParts) == 2 {
		num, _ := strconv.ParseFloat(frameRateParts[0], 64)
		den, _ := strconv.ParseFloat(frameRateParts[1], 64)
		if den > 0 {
			frameRate = num / den
		}
	}

	return StreamInfo{
		Codec:     codec,
		BitRate:   bitRate,
		Width:     width,
		Height:    height,
		FrameRate: frameRate,
		RtspURL:   camera.RtspURL,
	}, nil
}

// logStreamInfo логирует параметры потока.
func logStreamInfo(cameraID int, info StreamInfo) {
	log.Printf("Камера %d | Кодек: %s | Битрейт: %d | Разрешение: %dx%d | FPS: %.2f",
		cameraID, info.Codec, info.BitRate, info.Width, info.Height, info.FrameRate)
}

func configureFFmpegArgs(cameraID int, info StreamInfo) []string {
	args := []string{}

	decoder, hwaccel := selectDecoder(info.Codec)
	if hwaccel != "" {
		args = append(args, strings.Split(hwaccel, " ")...)
	}

	args = append(args,
		"-rtsp_transport", "tcp",
		"-i", info.RtspURL,
		"-rtbufsize", "20M",
		"-probesize", "2000000",
		"-analyzeduration", "500000",
	)

	if decoder != "copy" {
		args = append(args,
			"-c:v", decoder,
			"-preset", "fast",
			"-b:v", "2M",
		)
	} else {
		args = append(args, "-c:v", "copy")
	}

	args = append(args,
		"-g", "50",
		"-force_key_frames", "expr:gte(t,n_forced*2)",
		"-hls_time", "2",
		"-hls_list_size", "5",
		"-hls_flags", "append_list+delete_segments",
		"-an",
	)

	if info.BitRate > 4000000 {
		args = append(args, "-rtbufsize", "50M")
	}

	if info.FrameRate > 30 {
		args = append(args, "-r", "30")
	}

	args = append(args, storage.HLSPath+"/camera_"+strconv.Itoa(cameraID)+"/stream.m3u8")
	return args
}

// startFFmpeg запускает процесс FFmpeg и настраивает мониторинг ошибок.
func startFFmpeg(cameraID int, info StreamInfo) (*exec.Cmd, error) {
	args := configureFFmpegArgs(cameraID, info)
	cmd := exec.Command("ffmpeg", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении stderr: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ошибка при запуске ffmpeg: %v", err)
	}

	log.Printf("FFmpeg успешно запущен для камеры %d", cameraID)

	go monitorFFmpegErrors(cameraID, stderr)

	return cmd, nil
}

// monitorFFmpegErrors читает и логирует ошибки из stderr.
func monitorFFmpegErrors(cameraID int, stderr io.ReadCloser) {
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
}

func detectGPU() string {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("wmic", "path", "win32_VideoController", "get", "name").Output()
		if err != nil {
			return ""
		}
		output := strings.ToLower(string(out))
		if strings.Contains(output, "nvidia") {
			return "nvidia"
		} else if strings.Contains(output, "amd") || strings.Contains(output, "radeon") {
			return "amd"
		}
	} else if runtime.GOOS == "linux" {
		out, err := exec.Command("lspci").Output()
		if err != nil {
			return ""
		}
		output := strings.ToLower(string(out))
		if strings.Contains(output, "nvidia") {
			return "nvidia"
		} else if strings.Contains(output, "amd") || strings.Contains(output, "ati") {
			return "amd"
		}
	}
	return ""
}

func selectDecoder(codec string) (string, string) {
	if codec != "hevc" && codec != "h265" {
		return "copy", ""
	}

	gpu := detectGPU()

	switch gpu {
	case "nvidia":
		return "h264_nvenc", "-hwaccel cuda"
	case "amd":
		if runtime.GOOS == "windows" {
			return "libx264", "-hwaccel dxva2"
		}
		return "h264_vaapi", "-hwaccel vaapi"
	default:
		return "libx264", ""
	}
}
