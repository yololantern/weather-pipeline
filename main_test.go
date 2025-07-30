package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestMainProgram(t *testing.T) {
	// Capture standard output
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = stdout }()

	main()

	// Close the pipe and read the output
	w.Close()
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to copy output: %v", err)
	}
	r.Close()

	// Verify output
	expected := "Hello, World!\n"
	actual := buf.String()
	if actual != expected {
		t.Errorf("Expected %q but got %q", expected, actual)
	}
}
