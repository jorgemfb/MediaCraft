package config

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"

	"gopkg.in/ini.v1"
)

// Variables globales exportadas
var (
	Profiles            map[string][]string
	ProfileExts         map[string]string
	DefaultProfile      string
	OutputDir           string
	EnableNotifications bool
	TelegramToken       string
	TelegramChatID      string
)

// LoadProfiles carga perfiles y configuración desde el archivo INI
func LoadProfiles() error {
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
	Profiles = map[string][]string{}
	ProfileExts = map[string]string{}
	DefaultProfile = "telegram"
	OutputDir = ""
	EnableNotifications = false
	TelegramToken = ""
	TelegramChatID = ""
	// Leer configuración general
	if sec, err := cfg.GetSection("mediacraft"); err == nil {
		if sec.HasKey("default_profile") {
			DefaultProfile = sec.Key("default_profile").String()
		}
		if sec.HasKey("output_dir") {
			OutputDir = sec.Key("output_dir").String()
		}
		if sec.HasKey("notificaciones") {
			v := sec.Key("notificaciones").String()
			if v == "1" || v == "true" || v == "TRUE" || v == "True" {
				EnableNotifications = true
			}
		}
	}
	// Leer configuración de Telegram
	if sec, err := cfg.GetSection("telegram"); err == nil {
		if sec.HasKey("token") {
			TelegramToken = sec.Key("token").String()
		}
		if sec.HasKey("chat_id") {
			TelegramChatID = sec.Key("chat_id").String()
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
					ProfileExts[profileName] = v
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
			Profiles[profileName] = args
		}
	}
	return nil
}
