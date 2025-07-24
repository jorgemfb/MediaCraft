package order

import (
	"fmt"
	"mediacraft/decompress"
	"os"
	"path/filepath"
)

func OrderSeries(dir string) {
	// Descomprimir si es necesario (dummy)
	_ = decompress.Decompress(dir)
	// Lógica básica: crear carpeta "Temporada 1" y mover todo
	tempDir := filepath.Join(dir, "Temporada 1")
	os.MkdirAll(tempDir, 0755)
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if !f.IsDir() {
			os.Rename(filepath.Join(dir, f.Name()), filepath.Join(tempDir, f.Name()))
		}
	}
	fmt.Println("Archivos movidos a:", tempDir)
}
