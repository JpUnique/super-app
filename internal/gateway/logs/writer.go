package logs

import (
	"io"
	"os"
	"path/filepath"
	"sync"
)

// fileMultiWriter writes to stdout and a file (append).
type fileMultiWriter struct {
	mu    sync.Mutex
	file  *os.File
	multi io.Writer
	path  string
}

func NewFileMultiWriter(path string) (io.Writer, error) {
	w := &fileMultiWriter{path: path}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *fileMultiWriter) open() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.multi = io.MultiWriter(os.Stdout, f)
	return nil
}

func (w *fileMultiWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.multi.Write(p)
}

func (w *fileMultiWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		err := w.file.Close()
		w.file = nil
		return err
	}
	return nil
}
