package mocker

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	fslib "io/fs"
	"os/exec"
	"regexp"
	"sort"

	"github.com/joebjoe/go-mocker/internal/api"
	"github.com/joebjoe/go-mocker/internal/gofs"
)

type mocker struct{}

func New() api.Mocker {
	return &mocker{}
}

func getMethods(fs gofs.FS, req request) (methods []method, err error) {
	files, err := fslib.Glob(fs, "*.go")
	if err != nil {
		return nil, fmt.Errorf("failed to locate go files: %w", err)
	}

	pkgReg, err := regexp.Compile(fmt.Sprintf(patternFmtRegexPackageName, req.Package))
	if err != nil {
		return nil, fmt.Errorf("package '%s' is invalid", req.Package)
	}

	fileIsInPackage := pkgReg.Match

	sigReg, err := regexp.Compile(fmt.Sprintf(patternFmtRegexMethodSignature, req.Type))
	if err != nil {
		return nil, err
	}
	interfaceMethods := func(b []byte) (matches []string) {
		for _, m := range sigReg.FindAllStringSubmatch(string(b), -1) {
			matches = append(matches, m[1:]...)
		}
		return matches
	}

	var strMethods []string
	for _, fname := range files {
		fErr := func() error {
			if isTestFile(fname) {
				return nil
			}

			f, fErr := fs.Open(fname)
			if fErr != nil {
				return fmt.Errorf("failed to open: %w", err)
			}
			defer f.Close()

			b, rErr := io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("failed to read: %w", rErr)
			}

			if !fileIsInPackage(b) {
				return nil
			}

			strMethods = append(strMethods, interfaceMethods(b)...)

			return nil
		}()
		if fErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to process '%s': %w", fname, err))
		}
	}

	if err != nil {
		return nil, err
	}

	sort.Strings(strMethods)
	return parseMethods(strMethods, req.Package), nil
}

func (m *mocker) Generate(re api.RequestGET) (r io.Reader, err error) {
	req := request{RequestGET: re}

	fs, err := gofs.New(req.Module)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize file system: %w", err)
	}

	defer fs.Done()

	req.Methods, err = getMethods(fs, req)
	if err != nil {
		return nil, fmt.Errorf("failed to read package methods: %w", err)
	}

	if err = writeTemplateToWorkDir(fs, req); err != nil {
		return nil, fmt.Errorf("failed to generate %s %s: %w", req.Type, req.To, err)
	}

	goimports := exec.Command("goimports", "-local", req.Module, ".")
	goimports.Dir = fs.WorkDir()

	w, wErr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)

	goimports.Stdout = w
	goimports.Stderr = wErr

	if err = goimports.Run(); err != nil {
		b, _ := io.ReadAll(wErr)
		return nil, fmt.Errorf("failed to format files: %s %w", string(b), err)
	}

	cmdErrBytes, err := io.ReadAll(wErr)
	if len(cmdErrBytes) > 0 || err != nil {
		return nil, fmt.Errorf("failed to read goimports err output: %s %w", string(cmdErrBytes), err)
	}

	if len(cmdErrBytes) > 0 {
		return nil, fmt.Errorf("goimports was unsuccessful: %s", string(cmdErrBytes))
	}

	return w, nil
}
