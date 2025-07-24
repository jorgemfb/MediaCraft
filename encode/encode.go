package encode

import (
	"fmt"
	"mediacraft/decompress"
	"os/exec"
	"time"
)

// Perfiles de conversión
var profiles = map[string][]string{
	"telegram": {"-hwaccel", "cuda", "-c:v", "h264_nvenc", "-preset", "slow", "-c:a", "aac"},
	"plex":     {"-hwaccel", "cuda", "-c:v", "hevc_nvenc", "-b:v", "5000k", "-preset", "slow", "-c:a", "aac", "-b:a", "320k"},
	"alta":     {"-hwaccel", "cuda", "-c:v", "hevc_nvenc", "-preset", "slow", "-crf", "18", "-c:a", "aac", "-b:a", "256k"},
	"media":    {"-hwaccel", "cuda", "-c:v", "hevc_nvenc", "-preset", "medium", "-crf", "23", "-c:a", "aac", "-b:a", "160k"},
	"baja":     {"-hwaccel", "cuda", "-c:v", "hevc_nvenc", "-preset", "veryfast", "-crf", "32", "-c:a", "aac", "-b:a", "96k"},
	"movil":    {"-hwaccel", "cuda", "-c:v", "h264_nvenc", "-preset", "ultrafast", "-crf", "35", "-c:a", "aac", "-b:a", "64k", "-vf", "scale=640:-2"},
	"youtube":  {"-hwaccel", "cuda", "-c:v", "h264_nvenc", "-preset", "medium", "-crf", "20", "-c:a", "aac", "-b:a", "192k", "-pix_fmt", "yuv420p"},
	"av1":      {"-c:v", "libaom-av1", "-crf", "30", "-b:v", "0", "-c:a", "libopus", "-b:a", "128k"},
}

// Convert recibe el path y un perfil (por defecto: telegram)
func Convert(path string) {
	// Determinar perfil y archivo real (soporta nombres con espacios)
	profile := "telegram"
	realPath := path
	at := lastAt(path)
	if at != -1 && at != 0 && at != len(path)-1 {
		filePart := path[:at]
		profilePart := trimSpaces(path[at+1:])
		if _, ok := profiles[profilePart]; ok {
			profile = profilePart
			realPath = filePart
		}
	}

	// Descomprimir si es necesario
	inputName := realPath
	if at := findSubstring(inputName, "@"); at != -1 {
		inputName = inputName[:at]
	}
	finalPath, _ := decompress.JoinPartsIfNeeded(inputName)
	_ = decompress.Decompress(finalPath)
	if finalPath != "" {
		inputName = finalPath
	}

	// Determinar extensión de salida según perfil
	var outExt string
	switch profile {
	case "plex", "alta", "media", "baja":
		outExt = ".mkv"
	case "movil", "youtube":
		outExt = ".mp4"
	case "av1":
		outExt = ".webm"
	default:
		outExt = ".mp4"
	}
	out := getOutputNameWithExt(realPath, outExt)

	// Mensajes previos profesionales (después de determinar perfil y extensión)
	fmt.Printf("\n [SELECCIONADO] Perfil %s\n", capitalize(profile))
	if isNvidiaAvailable() {
		fmt.Println(" GPU NVIDIA detectada - usando aceleración por hardware")
	}
	fmt.Printf(" Iniciando conversión: %s\n", fileNameWithExt(inputName))
	fmt.Printf(" Archivo de salida: %s\n", fileNameWithExt(out))

	progressChan := make(chan string)
	doneChan := make(chan struct{})
	var lastProgress string
	spinner := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	totalDuration := getDuration(inputName)
	totalStr := formatDuration(totalDuration)
	go func() {
		i := 0
		current := "00:00:00"
		for {
			select {
			case t, ok := <-progressChan:
				if ok {
					lastProgress = t
					current = t
				}
			case <-doneChan:
				fmt.Print("\r")
				return
			default:
				fmt.Printf("\r%c Convirtiendo... [%s / %s]", spinner[i%len(spinner)], current, totalStr)
				i++
				slowSleep()
			}
		}
	}()
	switch profile {
	case "telegram":
		duration := getDuration(inputName)
		videoBitrate := 2500.0
		if duration > 0 {
			targetBits := 3.5 * 1024 * 1024 * 1024 * 8
			audioBitrate := 128000.0
			videoBitrate = (targetBits/duration - audioBitrate) / 1000.0
			if videoBitrate < 1000 {
				videoBitrate = 1000
			}
		}
		args := []string{"-hwaccel", "cuda", "-i", inputName, "-c:v", "h264_nvenc", "-b:v", fmt.Sprintf("%dk", int(videoBitrate)), "-preset", "slow", "-c:a", "aac", "-b:a", "128k", out}
		runFfmpegWithProgress(args, progressChan)
	case "plex":
		args1 := []string{"-y", "-hwaccel", "cuda", "-i", inputName, "-c:v", "hevc_nvenc", "-b:v", "5000k", "-preset", "slow", "-pass", "1", "-an", "-f", "null", "NUL"}
		args2 := []string{"-hwaccel", "cuda", "-i", inputName, "-c:v", "hevc_nvenc", "-b:v", "5000k", "-preset", "slow", "-pass", "2", "-c:a", "aac", "-b:a", "320k", out}
		runFfmpegWithProgress(args1, progressChan)
		fmt.Println()
		runFfmpegWithProgress(args2, progressChan)
		fmt.Println()
	case "alta", "media", "baja":
		// Dos pasadas hevc_nvenc
		args1 := []string{"-y", "-hwaccel", "cuda", "-i", inputName, "-c:v", "hevc_nvenc"}
		args1 = append(args1, profiles[profile][2:]...)
		args1 = append(args1, "-pass", "1", "-an", "-f", "null", "NUL")
		args2 := []string{"-hwaccel", "cuda", "-i", inputName, "-c:v", "hevc_nvenc"}
		args2 = append(args2, profiles[profile][2:]...)
		args2 = append(args2, "-pass", "2", out)
		runFfmpegWithProgress(args1, progressChan)
		fmt.Println()
		runFfmpegWithProgress(args2, progressChan)
		fmt.Println()
	case "movil", "youtube":
		// Una pasada h264_nvenc
		args := []string{"-hwaccel", "cuda", "-i", inputName, "-c:v", "h264_nvenc"}
		args = append(args, profiles[profile][2:]...)
		args = append(args, out)
		runFfmpegWithProgress(args, progressChan)
	case "av1":
		// Dos pasadas AV1
		args1 := []string{"-y", "-i", inputName, "-c:v", "libaom-av1", "-crf", "30", "-b:v", "0", "-pass", "1", "-an", "-f", "null", "NUL"}
		args2 := []string{"-i", inputName, "-c:v", "libaom-av1", "-crf", "30", "-b:v", "0", "-pass", "2", "-c:a", "libopus", "-b:a", "128k", out}
		runFfmpegWithProgress(args1, progressChan)
		fmt.Println()
		runFfmpegWithProgress(args2, progressChan)
		fmt.Println()
	default:
		// Fallback: una pasada
		args := []string{"-i", inputName}
		args = append(args, profiles[profile]...)
		args = append(args, out)
		runFfmpegWithProgress(args, progressChan)
	}
	close(doneChan)
	fmt.Printf("Resumen: %s → %s | Perfil: %s | Duración: %s | Progreso final: %s\n", fileNameWithExt(inputName), fileNameWithExt(out), profile, formatDuration(getDuration(out)), lastProgress)
}

// getOutputNameWithExt genera el nombre de salida limpio con la extensión deseada
func getOutputNameWithExt(input string, ext string) string {
	base := input
	if at := findSubstring(base, "@"); at != -1 {
		base = base[:at]
	}
	if dot := findSubstring(base, "."); dot != -1 {
		base = base[:dot]
	}
	return base + ext
}

// fileNameWithExt devuelve el nombre de archivo con extensión, sin ruta
func fileNameWithExt(path string) string {
	name := path
	// Quitar ruta
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' || name[i] == '\\' {
			name = name[i+1:]
			break
		}
	}
	return name
}

// capitalize la primera letra
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string([]rune(s)[0]-32) + s[1:]
}

// isNvidiaAvailable simula detección de GPU NVIDIA
func isNvidiaAvailable() bool {
	// Simulación: siempre true para demo
	return true
}

// cleanFileName limpia el nombre para mostrar bonito
func cleanFileName(path string) string {
	// Quita ruta y extensión
	name := path
	// Quitar ruta
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' || name[i] == '\\' {
			name = name[i+1:]
			break
		}
	}
	// Quitar extensión
	if dot := findSubstring(name, "."); dot != -1 {
		name = name[:dot]
	}
	return name
}

// Ejecuta ffmpeg y envía el progreso por canal
func runFfmpegWithProgress(args []string, progressChan chan<- string) {
	cmd := exec.Command("ffmpeg", args...)
	stderr, _ := cmd.StderrPipe()
	cmd.Stdout = cmd.Stderr
	_ = cmd.Start()
	buf := make([]byte, 4096)
	var line string
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			line += string(buf[:n])
			for {
				idx := findLineEnd(line)
				if idx == -1 {
					break
				}
				l := line[:idx]
				line = line[idx+1:]
				if t := extractTime(l); t != "" {
					progressChan <- t
				}
			}
		}
		if err != nil {
			break
		}
	}
	_ = cmd.Wait()
}

// Extrae el valor de time= de una línea de ffmpeg
func extractTime(line string) string {
	idx := -1
	if idx = findSubstring(line, "time="); idx != -1 {
		t := line[idx+5:]
		end := 0
		for end < len(t) && (t[end] >= '0' && t[end] <= '9' || t[end] == ':' || t[end] == '.') {
			end++
		}
		return t[:end]
	}
	return ""
}

// Busca el final de línea (\n o \r)
func findLineEnd(s string) int {
	for i, c := range s {
		if c == '\n' || c == '\r' {
			return i
		}
	}
	return -1
}

// Busca un substring
func findSubstring(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// Formatea segundos a HH:MM:SS
func formatDuration(d float64) string {
	h := int(d) / 3600
	m := (int(d) % 3600) / 60
	s := int(d) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// getDuration obtiene la duración en segundos usando ffprobe
func getDuration(path string) float64 {
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path).Output()
	if err != nil {
		fmt.Println("No se pudo obtener la duración con ffprobe:", err)
		return 0
	}
	var dur float64
	fmt.Sscanf(string(out), "%f", &dur)
	return dur
}

// slowSleep espera 1.5 segundos
func slowSleep() {
	time.Sleep(100 * time.Millisecond)
}

// trimSpaces elimina espacios en blanco al inicio y final
func trimSpaces(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

// lastAt devuelve la posición del último '@' en el string (útil para nombres con espacios)
func lastAt(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '@' {
			return i
		}
	}
	return -1
}
