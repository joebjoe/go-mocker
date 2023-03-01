package gofs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	fslib "io/fs"
	"os"
	"os/exec"
	"path"
)

type gofs string

func New(pkg string) (fslib.FS, error) {
	if err := exec.Command("go", "get", pkg).Run(); err != nil {
		return nil, fmt.Errorf("failed to go get '%s': %w", pkg, err)
	}

	cmd := exec.Command("go", "list", "-json", pkg)
	w, wErr := new(bytes.Buffer), new(bytes.Buffer)
	cmd.Stdout = w
	cmd.Stderr = wErr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed at go list: %w", err)
	}

	errBytes, err := io.ReadAll(wErr)
	if err != nil {
		return nil, fmt.Errorf("failed to read error: %w", err)
	}

	if len(errBytes) > 0 {
		return nil, fmt.Errorf("faild to get module metadata: %s", string(errBytes))
	}

	metadata := new(struct{ Dir string })
	if err := json.NewDecoder(w).Decode(metadata); err != nil {
		return nil, fmt.Errorf("failed to read go list output: %w", err)
	}

	return gofs(metadata.Dir), nil
}

func (fs gofs) Open(fname string) (fslib.File, error) {
	return os.Open(path.Join(string(fs), fname))
}
