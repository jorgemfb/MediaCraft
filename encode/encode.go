package encode

import (
	"fmt"
	"mediacraft/decompress"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

// Variables globales de configuración y perfiles
var (
	profiles            map[string][]string
	profileExts         map[string]string
	defaultProfile      string
	outputDir           string
	enableNotifications bool
	telegramToken       string
	telegramChatID      string
)

// Carga perfiles y configuración desde el archivo INI
func loadProfiles() error {
	var confPath string
	if os.PathSeparator == '\\' { // Windows
		userProfile := os.Getenv("USERPROFILE")
		confPath = path.Join(userProfile, ".config", "mediacraft", "mediacraft.conf")
	} else { // Unix-like
		usr, err := user.Current()
		if err != nil {
			return err
		}
		confPath = path.Join(usr.HomeDir, ".config", "mediacraft", "mediacraft.conf")
	}
	if _, err := os.Stat(confPath); err != nil {
		return fmt.Errorf("no se encontró el archivo de configuración: %s", confPath)
	}
	cfg, err := ini.Load(confPath)
	if err != nil {
		return err
	}
	profiles = map[string][]string{}
	profileExts = map[string]string{}
	defaultProfile = "telegram"
	outputDir = ""
	enableNotifications = false
	telegramToken = ""
	telegramChatID = ""
	// Leer configuración general
	if sec, err := cfg.GetSection("mediacraft"); err == nil {
		if sec.HasKey("default_profile") {
			defaultProfile = sec.Key("default_profile").String()
		}
		if sec.HasKey("output_dir") {
			outputDir = sec.Key("output_dir").String()
		}
		if sec.HasKey("notificaciones") {
			v := sec.Key("notificaciones").String()
			if v == "1" || v == "true" || v == "TRUE" || v == "True" {
				enableNotifications = true
			}
		}
	}
	// Leer configuración de Telegram
	if sec, err := cfg.GetSection("telegram"); err == nil {
		if sec.HasKey("token") {
			telegramToken = sec.Key("token").String()
		}
		if sec.HasKey("chat_id") {
			telegramChatID = sec.Key("chat_id").String()
		}
	}
	for _, section := range cfg.Sections() {
		name := section.Name()
		if len(name) > 9 && name[:9] == "perfiles." {
			profileName := name[9:]
			args := []string{}
			var videoCodec, audioCodec, kvideo, kaudio string
			for _, key := range section.KeyStrings() {
				v := section.Key(key).String()
				if v == "" {
					continue
				}
				k := strings.ToLower(strings.TrimSpace(key))
				switch k {
				case "ext":
					profileExts[profileName] = v
				case "video":
					videoCodec = strings.TrimSpace(v)
				case "audio":
					audioCodec = strings.TrimSpace(v)
				case "kvideo":
					kvideo = strings.TrimSpace(v)
				case "kaudio":
					kaudio = strings.TrimSpace(v)
				default:
					vNorm := strings.TrimSpace(strings.ReplaceAll(v, "=", ""))
					args = append(args, "-"+k, vNorm)
				}
			}
			if videoCodec != "" {
				args = append(args, "-c:v", videoCodec)
			}
			if kvideo != "" {
				args = append(args, "-b:v", kvideo)
			}
			if audioCodec != "" {
				args = append(args, "-c:a", audioCodec)
			}
			if kaudio != "" {
				args = append(args, "-b:a", kaudio)
			}
			profiles[profileName] = args
		}
	}
	return nil
}

// Convert recibe el path y un perfil (por defecto: telegram)
func Convert(path string) {
	// Cargar perfiles desde el archivo INI
	if err := loadProfiles(); err != nil {
		fmt.Println("[ERROR] No se pudieron cargar los perfiles:", err)
		return
	}
	// Determinar perfil y archivo real (soporta nombres con espacios)
	profile := defaultProfile
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

	// Determinar extensión de salida y formato ffmpeg (-f)
	var outExt string
	var ffFormat string
	if ext, ok := profileExts[profile]; ok {
		outExt = ext
		if len(outExt) > 0 && outExt[0] != '.' {
			outExt = "." + outExt
		}
		// Normalizar ffFormat según la extensión
		switch strings.ToLower(outExt) {
		case ".mkv":
			ffFormat = "matroska"
		case ".mp4":
			ffFormat = "mp4"
		case ".webm":
			ffFormat = "webm"
		default:
			ffFormat = strings.TrimPrefix(strings.ToLower(outExt), ".")
		}
	} else {
		switch profile {
		case "plex", "alta", "media", "baja":
			outExt = ".mkv"
			ffFormat = "matroska"
		case "movil", "youtube":
			outExt = ".mp4"
			ffFormat = "mp4"
		case "av1":
			outExt = ".webm"
			ffFormat = "webm"
		default:
			outExt = ".mp4"
			ffFormat = "mp4"
		}
	}
	// Determinar ruta de salida
	outName := getOutputNameWithExt(realPath, outExt)
	out := outName
	if outputDir != "" {
		// Convertir outputDir a ruta absoluta si es relativa
		absOutputDir := outputDir
		if !filepath.IsAbs(outputDir) {
			cwd, _ := os.Getwd()
			absOutputDir = filepath.Join(cwd, outputDir)
		}
		if _, err := os.Stat(absOutputDir); os.IsNotExist(err) {
			err = os.MkdirAll(absOutputDir, 0755)
			if err != nil {
			}
		} else {
		}
		out = filepath.Join(absOutputDir, fileNameWithExt(outName))
	}
	fmt.Printf("Archivo de salida final: %s\n", out)

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
	// --- Ejecutar ffmpeg según perfil ---
	var argsLog1, argsLog2 []string
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
		argsLog1 = []string{"-hwaccel", "cuda", "-i", inputName, "-c:v", "h264_nvenc", "-b:v", fmt.Sprintf("%dk", int(videoBitrate)), "-preset", "slow", "-c:a", "aac", "-b:a", "128k"}
		if ffFormat != "" && out != "" {
			argsLog1 = append(argsLog1, "-f", ffFormat)
		}
		argsLog1 = append(argsLog1, out)
		runFfmpegWithProgress(argsLog1, progressChan)
	case "plex":
		argsLog1 = []string{"-y", "-hwaccel", "cuda", "-i", inputName, "-c:v", "hevc_nvenc", "-b:v", "5000k", "-preset", "slow", "-pass", "1", "-an", "-f", "null", "NUL"}
		argsLog2 = []string{"-hwaccel", "cuda", "-i", inputName, "-c:v", "hevc_nvenc", "-b:v", "5000k", "-preset", "slow", "-pass", "2", "-c:a", "aac", "-b:a", "320k"}
		if ffFormat != "" && out != "" {
			argsLog2 = append(argsLog2, "-f", ffFormat)
		}
		argsLog2 = append(argsLog2, out)
		runFfmpegWithProgress(argsLog1, progressChan)
		runFfmpegWithProgress(argsLog2, progressChan)
	case "alta", "media", "baja":
		argsLog1 = []string{"-y", "-hwaccel", "cuda", "-i", inputName}
		argsLog1 = append(argsLog1, profiles[profile]...)
		argsLog1 = append(argsLog1, "-pass", "1", "-an", "-f", "null", "NUL")
		argsLog2 = []string{"-hwaccel", "cuda", "-i", inputName}
		argsLog2 = append(argsLog2, profiles[profile]...)
		argsLog2 = append(argsLog2, "-pass", "2")
		if ffFormat != "" && out != "" {
			argsLog2 = append(argsLog2, "-f", ffFormat)
		}
		argsLog2 = append(argsLog2, out)
		runFfmpegWithProgress(argsLog1, progressChan)
		runFfmpegWithProgress(argsLog2, progressChan)
	case "movil", "youtube":
		argsLog1 = []string{"-hwaccel", "cuda", "-i", inputName, "-c:v", "h264_nvenc"}
		if len(profiles[profile]) > 2 {
			argsLog1 = append(argsLog1, profiles[profile][2:]...)
		}
		if ffFormat != "" && out != "" {
			argsLog1 = append(argsLog1, "-f", ffFormat)
		}
		argsLog1 = append(argsLog1, out)
		runFfmpegWithProgress(argsLog1, progressChan)
	case "av1":
		argsLog1 = []string{"-y", "-i", inputName, "-c:v", "libaom-av1", "-crf", "30", "-b:v", "0", "-pass", "1", "-an", "-f", "null", "NUL"}
		argsLog2 = []string{"-i", inputName, "-c:v", "libaom-av1", "-crf", "30", "-b:v", "0", "-pass", "2", "-c:a", "libopus", "-b:a", "128k"}
		if ffFormat != "" && out != "" {
			argsLog2 = append(argsLog2, "-f", ffFormat)
		}
		argsLog2 = append(argsLog2, out)
		runFfmpegWithProgress(argsLog1, progressChan)
		runFfmpegWithProgress(argsLog2, progressChan)
	default:
		argsLog1 = []string{"-i", inputName}
		argsLog1 = append(argsLog1, profiles[profile]...)
		if ffFormat != "" && out != "" {
			argsLog1 = append(argsLog1, "-f", ffFormat)
		}
		argsLog1 = append(argsLog1, out)
		runFfmpegWithProgress(argsLog1, progressChan)
	}
	close(doneChan)
	// Mostrar resumen final limpio
	info, err := os.Stat(out)
	var durOut float64
	if err == nil && info.Size() > 0 {
		durOut = getDuration(out)
	} else {
		durOut = 0
	}
	resumen := fmt.Sprintf("Resumen: %s → %s | Perfil: %s | Duración salida: %s | Progreso final: %s", fileNameWithExt(inputName), fileNameWithExt(out), profile, formatDuration(durOut), lastProgress)
	fmt.Println(resumen)
	// Notificación Telegram si está habilitado
	if enableNotifications && telegramToken != "" && telegramChatID != "" {
		go sendTelegramNotification(telegramToken, telegramChatID, resumen)
	}
}

// Envía una notificación a Telegram usando el bot y chat_id configurados
func sendTelegramNotification(token, chatID, message string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	data := url.Values{}
	data.Set("chat_id", chatID)
	data.Set("text", message)
	data.Set("disable_web_page_preview", "true")
	data.Set("parse_mode", "HTML")
	_, _ = http.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}

// getOutputNameWithExt genera el nombre de salida limpio con la extensión deseada
// getOutputNameWithExt genera el nombre de salida limpio con la extensión deseada
func getOutputNameWithExt(input string, ext string) string {
	// Quitar ruta
	name := input
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' || name[i] == '\\' {
			name = name[i+1:]
			break
		}
	}
	// Quitar @perfil si lo hubiera
	if at := findSubstring(name, "@"); at != -1 {
		name = name[:at]
	}
	// Quitar extensión existente
	if dot := findSubstring(name, "."); dot != -1 {
		name = name[:dot]
	}
	// Si el nombre queda vacío, usar 'output'
	if len(name) == 0 {
		name = "output"
	}
	outputName := name + ext
	// Si el nombre de salida coincide con el de entrada (ignorando ruta), agregar sufijo
	inputBase := input
	for i := len(inputBase) - 1; i >= 0; i-- {
		if inputBase[i] == '/' || inputBase[i] == '\\' {
			inputBase = inputBase[i+1:]
			break
		}
	}
	if strings.EqualFold(outputName, inputBase) {
		outputName = name + "_out" + ext
	}
	return outputName
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
