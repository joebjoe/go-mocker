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

type gofs struct {
	pkgdir, workdir string
}

type FS interface {
	fslib.FS
	WorkDir() string
	Done() error
}

func New(pkg string) (FS, error) {
	wrkdir, err := os.MkdirTemp("", "*")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize working directory: %w", err)
	}

	newCmd := func(name string, args ...string) *exec.Cmd {
		cmd := exec.Command(name, args...)
		cmd.Dir = wrkdir
		// cmd.Stderr = os.Stderr
		return cmd
	}

	if err = newCmd("go", "mod", "init", "temp").Run(); err != nil {
		return nil, fmt.Errorf("failed to init go module in working directory: %w", err)
	}

	if err = newCmd("go", "get", pkg).Run(); err != nil {
		return nil, fmt.Errorf("failed to go get '%s': %w", pkg, err)
	}

	golist := newCmd("go", "list", "-json", pkg)
	stdOut, stdErr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	golist.Stdout, golist.Stderr = stdOut, stdErr

	if err = golist.Run(); err != nil {
		return nil, fmt.Errorf("failed to go list '%s': %w", pkg, err)
	}

	errBytes, err := io.ReadAll(stdErr)
	if err != nil {
		return nil, fmt.Errorf("failed to read 'go list' error: %w", err)
	}

	if len(errBytes) > 0 {
		return nil, fmt.Errorf("'go list' failed to run successfully: %s", string(errBytes))
	}

	golistResp := new(struct{ Dir string })
	if err := json.NewDecoder(stdOut).Decode(golistResp); err != nil {
		return nil, fmt.Errorf("failed to read 'go list' output: %w", err)
	}

	return &gofs{
		workdir: wrkdir,
		pkgdir:  golistResp.Dir,
	}, nil
}

func (fs *gofs) Open(fname string) (fslib.File, error) {
	return os.Open(path.Join(fs.pkgdir, fname))
}

func (fs *gofs) WorkDir() string {
	return fs.workdir
}

func (fs *gofs) Done() error {
	return os.RemoveAll(fs.workdir)
}
