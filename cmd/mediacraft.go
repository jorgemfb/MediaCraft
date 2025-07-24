package main

import (
	"flag"
	"fmt"
	"mediacraft/config"
	"mediacraft/encode"
	"mediacraft/order"
	"os"
)

var (
	convertFlag = flag.String("c", "", "Convertir archivo o carpeta de videos")
	orderFlag   = flag.String("o", "", "Ordenar archivos de series")
	versionFlag = flag.Bool("v", false, "Mostrar versión")
	helpFlag    = flag.Bool("h", false, "Mostrar ayuda")
)

const version = "v0.1.0"
const projectName = "Nostromo"
const author = "JorgeMFB"
const releaseDate = "24 de julio de 2025"

func main() {
	// Preprocesar flags largos para compatibilidad con el paquete flag
	var newArgs []string
	for i := 0; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "--help":
			newArgs = append(newArgs, "-h")
		case "--version":
			newArgs = append(newArgs, "-v")
		case "--convert":
			newArgs = append(newArgs, "-c")
			if i+1 < len(os.Args) {
				newArgs = append(newArgs, os.Args[i+1])
				i++ // saltar el valor
			}
		case "--order":
			newArgs = append(newArgs, "-o")
			if i+1 < len(os.Args) {
				newArgs = append(newArgs, os.Args[i+1])
				i++ // saltar el valor
			}
		default:
			if i != 0 { // omitir el nombre del ejecutable
				newArgs = append(newArgs, arg)
			}
		}
	}
	os.Args = append([]string{os.Args[0]}, newArgs...)
	flag.Parse()

	if *helpFlag {
		fmt.Printf("   MediaCraft CLI - Ayuda\n")
		fmt.Printf(" Uso: %s [flags]\n", projectName)
		fmt.Printf(" -c, --convert   Convertir archivo o carpeta de videos\n")
		fmt.Printf(" -o, --order     Ordenar archivos de series\n")
		fmt.Printf(" -v, --version   Mostrar versión\n")
		fmt.Printf(" -h, --help      Mostrar ayuda\n")
		os.Exit(0)
	}

	if *versionFlag {
		cyan := "\033[36m"
		reset := "\033[0m"
		fmt.Printf("%s   MediaCraft %s (%s) %s\n", cyan, version, projectName, reset)
		fmt.Printf("    Autor: %s\n", author)
		fmt.Printf("    Fecha: %s\n", releaseDate)
		os.Exit(0)
	}

	config.LoadConfig()

	if *convertFlag != "" {
		encode.Convert(*convertFlag)
		return
	}

	if *orderFlag != "" {
		order.OrderSeries(*orderFlag)
		os.Exit(0)
	}

	red := "\033[31m"
	reset := "\033[0m"
	fmt.Printf("%sNo se especificó ninguna acción.%s\n", red, reset)
	fmt.Printf("Use -h o --help para ver las opciones.\n")
	os.Exit(1)
}
