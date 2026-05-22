package utils

import (
	"bufio"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/google/go-jsonnet"
)

func GetLogLevelFromEnv() slog.Level {
	levelStr := os.Getenv("LOG_LEVEL")
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.Level(100) // Custom level higher than any standard level, so silent by default
	}
}

func LoadJsonFrom(path string, value any) {
	file, err := os.Open(path)
	if err != nil {
		log.Panicf("Failed opening file %s", path)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	decoder := json.NewDecoder(reader)
	err = decoder.Decode(value)
	if err != nil {
		log.Panicf("Failed reading+parsing file %s to %s: %s", path, reflect.TypeOf(value), err)
	}
}

func SaveJsonTo(path string, value any) {
	file, err := os.Create(path)
	if err != nil {
		log.Panicf("Failed opening file %s", path)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(value)
	if err != nil {
		log.Panicf("Failed writing+serializing file %s to %s: %s", path, reflect.TypeOf(value), err)
	}
}

func EvalJsonnetFrom(path string, value any) {
	vm := jsonnet.MakeVM()
	json_str, err := vm.EvaluateFile(path)
	if err != nil {
		log.Panicf("Failed evaluating JSONNET file %s: %s", path, err)
	}
	decoder := json.NewDecoder(strings.NewReader(json_str))
	err = decoder.Decode(value)
	if err != nil {
		log.Panicf("Failed reading+parsing file %s to %s: %s", path, reflect.TypeOf(value), err)
	}
}

func InterpretTildeHome(homedir string, path string) string {
	if path == "~" {
		return homedir
	} else if strings.HasPrefix(path, "~/") {
		return filepath.Join(homedir, path[2:])
	}
	return path
}
