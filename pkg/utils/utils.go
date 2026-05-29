package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/google/go-jsonnet"
	"golang.org/x/crypto/ssh"
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

	reader := bufio.NewReader(file)
	decoder := json.NewDecoder(reader)
	err = decoder.Decode(value)
	if err != nil {
		log.Panicf("Failed reading+parsing file %s to %s: %s", path, reflect.TypeOf(value), err)
	}

	if err = file.Close(); err != nil {
		log.Panic(err)
	}
}

func SaveJsonTo(path string, value any) {
	file, err := os.Create(path)
	if err != nil {
		log.Panicf("Failed opening file %s", path)
	}

	writer := bufio.NewWriter(file)

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(value)
	if err != nil {
		log.Panicf("Failed writing+serializing file %s to %s: %s", path, reflect.TypeOf(value), err)
	}

	if err = writer.Flush(); err != nil {
		log.Panic(err)
	}

	if err = file.Close(); err != nil {
		log.Panic(err)
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

func SSHRun(client *ssh.Client, command string) (int, string) {
	return SSHRunWithStdin(client, command, nil)
}

func InstallAgent(client *ssh.Client, agent []byte) (int, string) {
	return SSHRunWithStdin(client, "tee agent >/dev/null && chmod +x agent", &agent)
}

func SSHRunWithStdin(client *ssh.Client, command string, stdin_payload *[]byte) (int, string) {
	var exit_code int
	exit_code = 0

	session, err := client.NewSession()
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	if stdin_payload != nil {
		wg.Go(func() {
			stdin, err := session.StdinPipe()
			if err != nil {
				log.Fatal(err)
			}

			_, err = io.Copy(stdin, bytes.NewReader(*stdin_payload))
			if err != nil {
				log.Fatal(err)
			}

			if err = stdin.Close(); err != nil {
				log.Fatal(err)
			}
		})
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Panic(err)
	}

	if err = session.Run(command); err != nil {
		var exit_error *ssh.ExitError
		if errors.As(err, &exit_error) {
			exit_code = exit_error.ExitStatus()
		} else {
			log.Fatal(err)
		}
	}

	out, err := io.ReadAll(stdout)
	if err != nil {
		log.Panic(err)
	}

	if stdin_payload != nil {
		wg.Wait()
	}

	if err = session.Close(); err != nil {
		if err != io.EOF {
			log.Fatal(err)
		}
	}

	return exit_code, string(out)
}
