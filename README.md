# wiki2docx

Builds a Go CLI that fetches Wikipedia articles concurrently and saves each one as a `.docx` file.

The reason I built this tool is to generate dummy `.docx` files, which are required to unlock or download the actual files I need from certain websites.

## Features

- Fetch `N` random Wikipedia articles
- Read article titles from a `.txt` file (one title per line)
- Process articles concurrently with a worker pool
- Save each article as a separate `.docx` file
- Select Wikipedia language (`en`, `ru`, `de`, etc.)

## Requirements

- Go (version from `go.mod`: `1.25.5` or compatible)
- Internet access to the Wikipedia API
- Linux/macOS/Windows

## Installation and Build

### 1) Clone the repository

```bash
git clone https://github.com/w0ikid/wiki2docx.git
cd wiki2docx
```

### 2) Sync dependencies

```bash
go mod tidy
```

### 3) Build the binary

```bash
go build -o wiki2docx .
```

## Usage

There are 2 modes:

1. From an input file (`-input`)
2. Random articles (`-random`)

### Option A: From file

```bash
./wiki2docx -input titles.txt -out ./output -workers 5 -lang en
```

Example `titles.txt`:

```txt
Colonnaden
Caramelo (dog)
# comment line, ignored
Alan Turing
```

Input file rules:

- Empty lines are ignored
- Lines starting with `#` are treated as comments
- One line = one article title

### Option B: Random articles

```bash
# Uses aliases: -o for -out, -w for -workers
./wiki2docx -random 10 -o ./output -w 10 -lang ru
```

## CLI Flags

- `-input string`  
  Path to a `.txt` file with article titles. If set, `-random` is ignored.
- `-random int`  
  Number of random articles (default: `1`). Ensures unique titles.
- `-out string`, `-output`, `-o`  
  Output directory for `.docx` files (default: `./output`).
- `-workers int`, `-worker`, `-w`  
  Number of concurrent workers (default: `5`).
- `-rate int`  
  Global rate limit in requests per second (default: `10`). Set to `0` for no limit. Use this to avoid 429 errors when using many workers.
- `-lang string`  
  Wikipedia language (default: `en`), e.g. `ru`, `de`, `fr`.

## Output Files

- One `.docx` file is created per article
- File names are sanitized (unsafe characters are replaced with `_`)
- Files are saved to the directory from `-out`

## Full Example

```bash
go mod tidy
go build -o wiki2docx .
# Using short flags for convenience
./wiki2docx -random 50 -o ./output -w 20 -lang ru
ls ./output
```

## Project Structure

```text
.
├── main.go                # CLI entry point, flag parsing, worker pool
├── internal/wiki          # Wikipedia API integration
├── internal/docx          # DOCX generation
└── output                 # Results directory (created automatically)
```

## License

MIT (see `LICENSE`).
