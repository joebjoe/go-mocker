package mocker

import (
	"fmt"
	"strings"

	"github.com/joebjoe/go-mocker/internal/api"
)

type param struct {
	name, typ string
}

type Param interface {
	Name() string
	Type() string
	SetType(string)
}

type ParamList []Param

func (pl ParamList) String() string {
	b := new(strings.Builder)
	fmt.Fprint(b, "(")
	for i, p := range pl {
		if i > 0 {
			fmt.Fprint(b, ", ")
		}

		fmt.Fprint(b, p.Name())

		if p.Name() != "" && p.Type() != "" {
			fmt.Fprint(b, " ")
		}

		fmt.Fprint(b, p.Type())
	}
	fmt.Fprint(b, ")")

	return b.String()
}

func (p *param) Name() string     { return p.name }
func (p *param) Type() string     { return p.typ }
func (p *param) SetType(t string) { p.typ = t }

type method struct {
	Name           string
	RequestParams  ParamList
	ResponseParams ParamList
}

type request struct {
	api.RequestGET
	Methods []method `json:"-"`
}
