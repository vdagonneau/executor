package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
)

var commitHash string

func main() {
	filename := pflag.String("filename", "", "filename")
	pflag.Parse()

	args := pflag.Args()
	if len(args) == 0 {
		return
	}

	switch args[0] {
	case "version":
		printVersion(os.Stdout)
	case "copy":
		if *filename == "" {
			log.Fatal("copy requires --filename")
		}
		if err := copyFromBase64(os.Stdin, *filename); err != nil {
			log.Fatalf("copy failed: %s", err)
		}
	default:
		log.Fatalf("unknown action: %s", args[0])
	}
}

func printVersion(stdout io.Writer) {
	fmt.Fprint(stdout, commitHash)
}

func copyFromBase64(stdin io.Reader, filename string) error {
	dir := filepath.Dir(filename)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(filename)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	decoder := base64.NewDecoder(base64.StdEncoding, stdin)
	if _, err = io.Copy(tmp, decoder); err != nil {
		_ = tmp.Close()
		return err
	}

	if err = tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, filename)
}
