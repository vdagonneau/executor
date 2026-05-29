package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintVersionWritesCommitHash(t *testing.T) {
	original := commitHash
	t.Cleanup(func() {
		commitHash = original
	})
	commitHash = "test-commit"

	var stdout bytes.Buffer
	printVersion(&stdout)

	if stdout.String() != "test-commit" {
		t.Fatalf("version output mismatch: got %q want %q", stdout.String(), "test-commit")
	}
}

func TestCopyFromBase64WritesDecodedData(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "payload.bin")
	payload := []byte("hello\nworld\x00")
	input := base64.StdEncoding.EncodeToString(payload)

	if err := copyFromBase64(strings.NewReader(input), filename); err != nil {
		t.Fatalf("copyFromBase64 failed: %s", err)
	}

	got, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed reading copied file: %s", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("copied data mismatch: got %q want %q", got, payload)
	}
}

func TestCopyFromBase64DoesNotOverwriteOnInvalidBase64(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "payload.bin")
	original := []byte("original")
	if err := os.WriteFile(filename, original, 0644); err != nil {
		t.Fatalf("failed writing original file: %s", err)
	}

	err := copyFromBase64(strings.NewReader("not-base64!"), filename)
	if err == nil {
		t.Fatal("expected invalid base64 error")
	}

	got, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed reading original file: %s", err)
	}
	if string(got) != string(original) {
		t.Fatalf("invalid input overwrote file: got %q want %q", got, original)
	}
}
