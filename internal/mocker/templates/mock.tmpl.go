package mock
{{$mockType := .Type}}
type I{{$mockType}} interface {
	{{- range .Methods }}
	{{.}}
	{{- end}}
}

type Mock{{$mockType}} struct {
{{- range .Methods }}
	{{ $fname := FuncName .}}
	{{- $calledWithType := CalledWithType .}}
	{{$fname}}CalledTimes int
	{{- if ne $calledWithType ""}}
	{{$fname}}CalledWith []{{$calledWithType}}
	{{- end}}
	{{- $isVoid := ReturnsVoid .}}
	{{- if not $isVoid}}
	Mock{{$fname}} func{{TrimPrefix . $fname}}
	{{- end}}
{{- end}}
}

{{- range .Methods}}
{{$fname := FuncName .}}
func (m *Mock{{$mockType}}) {{.}} {
	m.{{$fname}}CalledTimes++
	{{$calledWithAppend := CalledWithAppend .}}
	{{- if ne $calledWithAppend ""}}
	m.{{$fname}}CalledWith = append(m.{{$fname}}CalledWith, {{$calledWithAppend}})
	{{- end}}
	{{- $returnsVoid := ReturnsVoid .}}
	{{- if not $returnsVoid}}
	return m.Mock{{$fname}}({{Params .}})
	{{- end}}
} 
{{- end}}

{{- range .Methods }}
{{GetInputStruct .}}
{{- end}}