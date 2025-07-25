package order

import (
	"fmt"
	"mediacraft/decompress"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func OrderSeries(dir string) {
	// Verde: \033[32m, Azul: \033[34m, Amarillo: \033[33m, Reset: \033[0m
	green := "\033[32m"
	blue := "\033[34m"
	yellow := "\033[33m"
	reset := "\033[0m"
	fmt.Printf("%s  Leyendo archivos de la carpeta:%s %s\n", blue, reset, dir) // nf-fa-tasks
	// Descomprimir si es necesario
	extracted, err := decompress.DecompressAuto(dir)
	if err == nil && len(extracted) > 0 && (len(extracted) != 1 || extracted[0] != dir) {
		fmt.Printf("%s  Archivos comprimidos detectados y extraídos a temporal:%s\n", yellow, reset)
		// Si se extrajo, usar la carpeta temporal del primer archivo extraído
		dir = filepath.Dir(extracted[0])
	}
	files, _ := os.ReadDir(dir)
	fmt.Printf("%s  %d archivos encontrados. Detectando temporadas...%s\n", yellow, len(files), reset) // nf-fa-file_text
	temporadas := make(map[string][]string)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		temp := detectSeason(name)
		if temp == 0 {
			temp = 1 // Si no se detecta, poner en Temporada 1
		}
		key := fmt.Sprintf("Temporada %d", temp)
		temporadas[key] = append(temporadas[key], name)
	}
	fmt.Printf("\n%s  Creando carpetas y moviendo archivos...%s\n", blue, reset) // nf-fa-folder
	for key, files := range temporadas {
		tempDir := filepath.Join(dir, key)
		os.MkdirAll(tempDir, 0755)
		for _, fname := range files {
			os.Rename(filepath.Join(dir, fname), filepath.Join(tempDir, fname))
		}
	}
	fmt.Printf("\n%s  Ordenación de series completada.%s\n", green, reset) // nf-fa-check
}

// detectSeason intenta extraer el número de temporada de un nombre de archivo
func detectSeason(name string) int {
	name = strings.ToLower(name)
	// Patrones comunes: S01, S1, T01, T1, 1x01, 2x05, season 1, temp1, temporada 1, etc.
	patterns := []string{
		`s(\d{1,2})`,                // S01, S1
		`t(\d{1,2})`,                // T01, T1
		`(\d{1,2})x\d{2,3}`,         // 1x01, 2x05
		`season[ ._-]?(\d{1,2})`,    // season 1
		`temp[ ._-]?(\d{1,2})`,      // temp1
		`temporada[ ._-]?(\d{1,2})`, // temporada 1
	}
	for _, pat := range patterns {
		re := regexp.MustCompile(pat)
		match := re.FindStringSubmatch(name)
		if len(match) > 1 {
			return atoiSafe(match[1])
		}
	}
	return 0
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func atoiSafe(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			break
		}
		n = n*10 + int(s[i]-'0')
	}
	return n
}
