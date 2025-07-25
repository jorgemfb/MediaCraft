package decompress

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// DecompressAuto: descomprime cualquier archivo comprimido (zip, rar, 7z, tar, gz, etc.)
// o multi-volumen, en una carpeta temporal. Devuelve los paths extraídos.
func DecompressAuto(path string) ([]string, error) {
	// Si es carpeta, buscar archivos comprimidos dentro
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var filesToExtract []string
	if info.IsDir() {
		err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if isCompressed(p) {
				filesToExtract = append(filesToExtract, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		if isCompressed(path) {
			filesToExtract = append(filesToExtract, path)
		}
	}
	if len(filesToExtract) == 0 {
		return []string{path}, nil // No hay nada que descomprimir
	}
	var allExtracted []string
	for _, f := range filesToExtract {
		joined, _ := JoinPartsIfNeeded(f)
		tmpDir, err := os.MkdirTemp("", "mediacraft_unzip_")
		if err != nil {
			return nil, err
		}
		err = decompressWith7z(joined, tmpDir)
		if err != nil {
			return nil, err
		}
		// Listar archivos extraídos
		filepath.WalkDir(tmpDir, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				allExtracted = append(allExtracted, p)
			}
			return nil
		})
	}
	return allExtracted, nil
}

// decompressWith7z ejecuta 7z x archivo -o<destino>
func decompressWith7z(archive, dest string) error {
	cmd := exec.Command("7z", "x", archive, "-o"+dest, "-y")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// isCompressed detecta si el archivo es comprimido o multi-volumen
func isCompressed(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	compressed := []string{".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz", ".lz", ".lzma", ".z01", ".001", ".part1", ".part01"}
	for _, e := range compressed {
		if strings.HasSuffix(ext, e) {
			return true
		}
	}
	// Detectar multi-volumen por nombre
	multi := regexp.MustCompile(`(?i)\.(part|vol|disk)?0*1(\.|$)`)
	return multi.MatchString(path)
}

// atoiSafe convierte string a int (solo dígitos)
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

// Detecta y une partes de archivos partidos (.001, .part01, .z01, etc.) en un archivo temporal único
func JoinPartsIfNeeded(path string) (string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	// Detectar patrón de parte
	patterns := []string{
		`(?i)(.+)\.part0*([1-9][0-9]*)\.(rar|7z|zip|tar|gz|bz2|lz|lzma|xz)$`,
		`(?i)(.+)\.0*([1-9][0-9]*)$`,
		`(?i)(.+)\.z0*([1-9][0-9]*)$`,
	}
	var matches [][]string
	var re *regexp.Regexp
	for _, pat := range patterns {
		re = regexp.MustCompile(pat)
		if m := re.FindStringSubmatch(base); m != nil {
			matches = append(matches, m)
			break
		}
	}
	if len(matches) == 0 {
		return path, nil // No es multi-volumen
	}
	m := matches[0]
	prefix := m[1]
	ext := ""
	if len(m) > 3 {
		ext = "." + m[3]
	}
	// Buscar todas las partes
	files, _ := os.ReadDir(dir)
	var parts []string
	for _, f := range files {
		fname := f.Name()
		for _, pat := range patterns {
			re = regexp.MustCompile(pat)
			if mm := re.FindStringSubmatch(fname); mm != nil && mm[1] == prefix {
				parts = append(parts, fname)
			}
		}
	}
	if len(parts) <= 1 {
		return path, nil // Solo una parte
	}
	// Ordenar partes por número
	type partInfo struct {
		name  string
		index int
	}
	var partList []partInfo
	for _, fname := range parts {
		for _, pat := range patterns {
			re = regexp.MustCompile(pat)
			if mm := re.FindStringSubmatch(fname); mm != nil && mm[1] == prefix {
				idx := atoiSafe(mm[2])
				partList = append(partList, partInfo{fname, idx})
			}
		}
	}
	// Ordenar por idx
	for i := 0; i < len(partList)-1; i++ {
		for j := i + 1; j < len(partList); j++ {
			if partList[i].index > partList[j].index {
				partList[i], partList[j] = partList[j], partList[i]
			}
		}
	}
	// Unir partes en archivo temporal
	tmpFile, err := os.CreateTemp("", "mediacraft_joined_*"+ext)
	if err != nil {
		return path, err
	}
	defer tmpFile.Close()
	for _, p := range partList {
		f, err := os.Open(filepath.Join(dir, p.name))
		if err != nil {
			return path, err
		}
		_, err = ioCopy(tmpFile, f)
		f.Close()
		if err != nil {
			return path, err
		}
	}
	return tmpFile.Name(), nil
}

// ioCopy es como io.Copy pero sin importar io
func ioCopy(dst *os.File, src *os.File) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			wn, werr := dst.Write(buf[:n])
			total += int64(wn)
			if werr != nil {
				return total, werr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return total, err
		}
	}
	return total, nil
}
