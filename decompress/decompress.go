package decompress

import (
	"os/exec"
)

// Descomprime un archivo usando 7z
func Decompress(path string) error {
	cmd := exec.Command("7z", "x", path)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// Detecta y une partes de archivos partidos (dummy)
func JoinPartsIfNeeded(path string) (string, error) {
	// Aquí iría la lógica real
	return path, nil
}
