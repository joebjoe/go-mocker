package mocker

import (
	"embed"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/joebjoe/go-mocker/internal/gofs"
)

const (
	patternFmtRegexPackageName     = `package %s`
	patternFmtRegexMethodSignature = `(?m)^func \([a-zA-Z][a-zA-Z0-9_]* \*?%s\) ([A-Z][a-zA-Z0-9_]+.*){$`

	patternCaptureSymbolName     = `([A-Z][a-zA-Z0-9_]*)`
	patternCaptureRequestParams  = `\((.+)\)`
	patternCaptureResponseParams = `\(?([^\)]+)\)?`

	patternExportedType = `(?:[\s\[\]\(\)\*])([A-Z][a-zA-Z0-9_]*)`
)

var (
	reSymbolName   = regexp.MustCompile(patternCaptureSymbolName)
	reVoidVoid     = regexp.MustCompile(`^` + patternCaptureSymbolName + `\(\)$`)
	reReqResp      = regexp.MustCompile(`(?m)^` + patternCaptureSymbolName + patternCaptureRequestParams + `\s*` + patternCaptureResponseParams + `$`)
	reVoidResp     = regexp.MustCompile(`(?m)^` + patternCaptureSymbolName + `\(\)\s*` + patternCaptureResponseParams + `$`)
	reReqVoid      = regexp.MustCompile(`(?m)^` + patternCaptureSymbolName + patternCaptureRequestParams + `$`)
	reExportedType = regexp.MustCompile(patternExportedType)

	//go:embed "templates"
	tmplFS embed.FS

	isTestFile = regexp.MustCompile(`_test.go$`).MatchString
	ifaceTmpl  = template.Must(
		template.New("").
			Funcs(template.FuncMap{
				"IFacePackageName": func(pkg, typ string) string {
					return strings.ToLower(fmt.Sprintf("%s%siface", pkg, typ))
				},
				"IFaceTypeName": func(pkg, typ string) string {
					return fmt.Sprintf("%s%sAPI", wordToTitleSimple(pkg), wordToTitleSimple(typ))
				},
				"MockName": func(pkg, typ string) string {
					return fmt.Sprintf("%s%s", wordToTitleSimple(pkg), wordToTitleSimple(typ))
				},
				"ToInputFieldDefinitions": func(params []Param) string {
					// populate all implied types
					for i := len(params) - 1; i > 0; i-- {
						if params[i-1].Type() == "" {
							params[i-1].SetType(params[i].Type())
						}
					}

					b := new(strings.Builder)

					for _, p := range params {
						typ := p.Type()
						if strings.HasPrefix(typ, "...") {
							typ = "[]" + typ[3:]
						}
						fmt.Fprintf(b, "\t%s %s\n", wordToTitleSimple(p.Name()), typ)
					}

					return b.String()
				},
				"ToInputInstance": func(params []Param, mockName string) string {
					if len(params) == 1 {
						return params[0].Name()
					}

					// populate all implied types
					for i := len(params) - 1; i > 0; i-- {
						if params[i-1].Type() == "" {
							params[i-1].SetType(params[i].Type())
						}
					}

					b := new(strings.Builder)

					fmt.Fprintf(b, "%sInput{\n", mockName)

					for _, p := range params {
						fmt.Fprintf(b, "\t%s: %s,\n", wordToTitleSimple(p.Name()), p.Name())
					}

					fmt.Fprint(b, "}")

					return b.String()
				},
				"ToParamNameList": func(params []Param) string {
					pnames := make([]string, len(params))
					for i, p := range params {
						pnames[i] = p.Name()
					}
					return strings.Join(pnames, ", ")
				},
			}).ParseFS(tmplFS, "templates/*.tmpl.go"))
)

func writeTemplateToWorkDir(fs gofs.FS, req request) error {
	fname := strings.ToLower(string(req.To))
	f, err := os.Create(path.Join(fs.WorkDir(), fname+".go"))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	return ifaceTmpl.ExecuteTemplate(f, fname+".tmpl.go", req)
}

func parseMethods(strMethods []string, pkg string) (methods []method) {
	var reMatches = func(s []string) ([]string, bool) {
		if len(s) > 0 {
			return s[1:], true
		}
		return nil, false
	}

	for _, sMthd := range strMethods {
		sMthd = strings.TrimSpace(sMthd)

		if matches, ok := reMatches(reVoidVoid.FindStringSubmatch(sMthd)); ok {
			methods = append(methods, method{
				Name: matches[0],
			})
			continue
		}
		if matches, ok := reMatches(reReqResp.FindStringSubmatch(sMthd)); ok {
			methods = append(methods, method{
				Name:           matches[0],
				RequestParams:  parseRequestParams(matches[1], pkg),
				ResponseParams: parseResponseParams(matches[2], pkg),
			})
			continue
		}

		if matches, ok := reMatches(reVoidResp.FindStringSubmatch(sMthd)); ok {
			methods = append(methods, method{
				Name:           matches[0],
				ResponseParams: parseResponseParams(matches[1], pkg),
			})
			continue
		}

		if matches, ok := reMatches(reReqVoid.FindStringSubmatch(sMthd)); ok {
			methods = append(methods, method{
				Name:          matches[0],
				RequestParams: parseRequestParams(matches[1], pkg),
			})
			continue
		}
	}

	return
}

func parseRequestParams(paramList, pkg string) (params []Param) {
	pList := strings.Split(paramList, ",")
	for _, p := range pList {
		p := strings.TrimSpace(p)
		pparts := strings.Split(p, " ")
		var typ string
		if len(pparts) > 1 {
			typ = pparts[1]

			typ = reExportedType.ReplaceAllStringFunc(typ, func(s string) string {
				return reSymbolName.ReplaceAllString(s, pkg+".$1")
			})
		}

		params = append(params, &param{name: pparts[0], typ: typ})
	}

	return params
}

func parseResponseParams(paramList, pkg string) []Param {
	pList := strings.Split(paramList, ",")

	var (
		params      = make([]Param, len(pList))
		namedParams bool
		ch          = make(chan struct{})
		mux         = new(sync.RWMutex)
		wg          = new(sync.WaitGroup)
	)

	for i, p := range pList {
		wg.Add(1)

		i, p := i, p
		p = strings.TrimSpace(p)
		pparts := strings.Split(p, " ")

		namedParams = namedParams || len(pparts) > 1

		go func(wg *sync.WaitGroup, mux *sync.RWMutex, ch <-chan struct{}, namedParams *bool, params *[]Param, pparts []string, i int) {
			defer wg.Done()
			defer mux.Unlock()
			<-ch
			mux.Lock()

			var typ string
			if !*namedParams {
				(*params)[i] = &param{typ: reExportedType.ReplaceAllStringFunc(pparts[0], func(s string) string {
					return reSymbolName.ReplaceAllString(s, pkg+".$1")
				})}

				return
			}

			if len(pparts) > 1 {
				typ = pparts[1]

				typ = reExportedType.ReplaceAllStringFunc(typ, func(s string) string {
					return reSymbolName.ReplaceAllString(s, pkg+".$1")
				})
			}

			(*params)[i] = &param{name: pparts[0], typ: typ}
		}(wg, mux, ch, &namedParams, &params, pparts, i)
	}
	close(ch)
	wg.Wait()

	return params
}

func wordToTitleSimple(word string) string {
	if len(word) <= 1 {
		return strings.ToUpper(word)
	}
	return strings.ToUpper(word[:1]) + word[1:]
}
