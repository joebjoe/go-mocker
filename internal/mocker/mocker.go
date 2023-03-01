package mocker

import (
	"embed"
	"errors"
	"fmt"
	"io"
	fslib "io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/joebjoe/go-mocker/internal/gofs"
)

const (
	fmtRegexPackageName     = `package %s`
	fmtRegexMethodSignature = `(?m)^func \([a-zA-Z][a-zA-Z0-9_]* \*?%s\) ([A-Z][a-zA-Z0-9_]+.*){$`
)

//go:embed "templates"
var tmplFS embed.FS

var isTestFile = regexp.MustCompile(`_test.go$`).MatchString
var ifaceTmpl = template.Must(
	template.New("").
		Funcs(template.FuncMap{
			"CalledWithAppend": getCalledWithAppend,
			"CalledWithType":   getCalledWithType,
			"FuncName":         funcName,
			"GetInputStruct":   getInputStruct,
			"Params":           getParams,
			"ReturnsVoid":      returnsVoid,
			"ToLower":          strings.ToLower,
			"TrimPrefix":       strings.TrimPrefix,
		}).ParseFS(tmplFS, "templates/mock.tmpl.go"))

type Request struct {
	Module  string
	Package string
	Type    string
	Methods []string `json:"-"`
}

func FindMethods(req Request) (methods []string, err error) {
	fs, err := gofs.New(req.Module)
	if err != nil {
		panic(err)
	}

	files, err := fslib.Glob(fs, "*.go")
	if err != nil {
		panic(err)
	}

	fmt.Println(files)
	pkgReg, err := regexp.Compile(fmt.Sprintf(fmtRegexPackageName, req.Package))
	if err != nil {
		return nil, err
	}
	isInPackage := pkgReg.Match

	sigReg, err := regexp.Compile(fmt.Sprintf(fmtRegexMethodSignature, req.Type))
	if err != nil {
		return nil, err
	}
	interfaceMethods := func(b []byte) (matches []string) {
		for _, m := range sigReg.FindAllStringSubmatch(string(b), -1) {
			matches = append(matches, m[1:]...)
		}
		return matches
	}

	for _, fname := range files {
		fErr := func() error {
			if isTestFile(fname) {
				return nil
			}

			f, fErr := fs.Open(fname)
			if fErr != nil {
				return fmt.Errorf("failed to open '%s': %w", fname, err)
			}
			defer f.Close()

			b, rErr := io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("failed to read '%s': %w", fname, rErr)
			}

			if !isInPackage(b) {
				return nil
			}

			methods = append(methods, interfaceMethods(b)...)

			return nil
		}()
		if fErr != nil {
			err = errors.Join(err, fErr)
		}
	}

	if err != nil {
		return nil, err
	}

	sort.Strings(methods)
	return methods, nil
}

func Generate(req Request) error {
	// w := bytes.NewBuffer(nil)

	return ifaceTmpl.ExecuteTemplate(os.Stdout, "mock.tmpl.go", req)
}

const captureFuncName = `^([A-Z][a-zA-Z0-9_]+)`

var reCaptureFuncName = regexp.MustCompile(captureFuncName)

const captureRequestParams = `\((.+)\)\s+.+$`

var reCaptureRequestParams = regexp.MustCompile(captureRequestParams)

var returnsVoid = func() func(string) bool {
	re := regexp.MustCompile(`\(.*\)\s.+$`)
	return func(sig string) bool {
		return !re.MatchString(sig)
	}
}()

func funcName(sig string) string { return reCaptureFuncName.FindStringSubmatch(sig)[1] }

func getCalledWithType(sig, pkg string) string {
	matches := reCaptureRequestParams.FindStringSubmatch(sig)
	if len(matches) == 0 {
		return ""
	}

	paramList := strings.Split(matches[1], ",")
	if len(paramList) > 1 {
		return funcName(sig) + "Input"
	}

	t := strings.Split(strings.TrimSpace(paramList[0]), " ")[1]
	return strings.Replace(t, "...", "[]", 1)
}

func getCalledWithAppend(sig string) string {
	matches := reCaptureRequestParams.FindStringSubmatch(sig)
	if len(matches) == 0 {
		return ""
	}

	params := splitParams(matches[1])
	if len(params) == 1 {
		p, _ := splitParam(params[0])
		return p
	}
	b := new(strings.Builder)
	fmt.Fprintf(b, "%sInput{\n", funcName(sig))

	lpad := 0
	for _, param := range params {
		p, _ := splitParam(param)
		if len(p) > lpad {
			lpad = len(p)
		}
	}

	for _, param := range params {
		p, _ := splitParam(param)
		fmt.Fprintf(b, "\t\t%s%s: %"+fmt.Sprint(lpad)+"s,\n", strings.ToUpper(string(p[0])), p[1:], p)
	}
	fmt.Fprint(b, "\t}")
	return b.String()
}

func getInputStruct(sig string) string {
	matches := reCaptureRequestParams.FindStringSubmatch(sig)
	if len(matches) == 0 {
		return ""
	}

	params := splitParams(matches[1])
	if len(params) == 1 {
		return ""
	}

	b := new(strings.Builder)
	fmt.Fprintf(b, "type %sInput struct {\n", funcName(sig))

	lpad := 0
	for _, param := range params {
		p, _ := splitParam(param)
		if len(p) > lpad {
			lpad = len(p)
		}
	}

	for _, param := range params {
		p, t := splitParam(param)
		msg := "\t%s%s %" + fmt.Sprint(lpad) + "s\n"
		t = strings.Replace(t, "...", "[]", 1)
		fmt.Fprintf(b, msg, strings.ToUpper(string(p[0])), p[1:], t)
	}
	fmt.Fprint(b, "}")

	return b.String()
}

func getParams(sig string) string {
	matches := reCaptureRequestParams.FindStringSubmatch(sig)
	if len(matches) == 0 {
		return ""
	}

	paramList := splitParams(matches[1])

	params := make([]string, len(paramList))
	for i, param := range paramList {
		p, t := splitParam(param)
		if isVariadic(t) {
			p = p + "..."
		}
		params[i] = p
	}
	return strings.Join(params, ", ")
}

func isVariadic(t string) bool { return strings.HasPrefix(t, "...") }
func splitParams(s string) []string {
	return strings.Split(strings.TrimSpace(s), ",")
}
func splitParam(s string) (p, t string) {
	parts := strings.Split(strings.TrimSpace(s), " ")
	return parts[0], parts[1]
}
