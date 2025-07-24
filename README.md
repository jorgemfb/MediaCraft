# Mediacraft CLI

Herramienta de línea de comandos en Go para automatizar la descompresión, conversión y organización de archivos multimedia en Windows 11.

## Estructura del Proyecto

```
mediacraft/
│
├── cmd/
│   └── mediacraft.go         // Entrada principal y parsing de flags
├── decompress/
│   └── decompress.go         // Lógica de descompresión y unión de partes
├── encode/
│   └── encode.go             // Lógica de conversión con ffmpeg y perfiles
├── order/
│   └── order.go              // Lógica de ordenado de series
├── config/
│   └── config.go             // Manejo de archivos .conf
├── utils/
│   └── utils.go              // Funciones auxiliares (detección de partes, validaciones, etc)
├── go.mod
└── README.md
```

## Funcionalidades Básicas

### 1. Conversión de archivos de vídeo
- Usa ffmpeg (debe estar en el PATH).
- Soporta archivos individuales o carpetas.
- Descomprime archivos comprimidos (incluyendo partidos) usando 7z.
- Perfiles de conversión: Telegram, Plex, Alta Calidad, Media Calidad, Baja Calidad, Dispositivos Móviles, Youtube, AV1.
- Detecta automáticamente la pista de audio en español.
- Usa GPU Nvidia si está disponible.

### 2. Ordenar archivos de vídeo (series)
- Detecta temporadas y crea carpetas "Temporada 1", "Temporada 2", etc.
- Mueve los archivos a su carpeta correspondiente.
- Descomprime archivos comprimidos (incluyendo partidos) usando 7z.

### 3. Descompresión
- Soporta `.zip`, `.rar`, `.7z`, `.tar.gz`, `.part1.rar`, `.001`, etc.
- Une partes automáticamente antes de descomprimir.
- Usa 7z para todo.

### 4. Configuración
- Usa un archivo `.conf` para rutas, tokens, chat de Telegram, rutas de herramientas, etc.

### 5. Flags del CLI
- `-c` / `--convert`   → Conversión de archivos
- `-o` / `--order`     → Ordenar series
- `-v` / `--version`   → Versión
- `-h` / `--help`      → Ayuda

---

## Requisitos
- Go 1.20+
- ffmpeg y 7z en el PATH
- Windows 11

## Uso básico

```sh
mediacraft -c carpeta_o_archivo [opciones]
mediacraft -o carpeta_de_series
```

---

Este proyecto se irá ampliando y mejorando según las necesidades.
