package mock

{{$mockName := MockName .Package .Type}}
// {{$mockName}} ...
type {{$mockName}} struct {
{{- range .Methods}}

	{{.Name}}CalledTimes int
	{{.Name}}CalledWith []{{- if gt (len .RequestParams) 1 -}}{{.Name}}Input{{- else}}{{ (index .RequestParams 0).Type }}{{- end}}
	Mock{{.Name}} func{{.RequestParams}} {{.ResponseParams}}
{{- end}}
}

{{- range .Methods}}
{{ if gt (len .RequestParams) 1 -}}
// {{.Name}}Input ...
type {{.Name}}Input struct {
	{{ToInputFieldDefinitions .RequestParams}}
}
{{ end}}

// {{.Name}} ...
func (m *{{$mockName}}) {{.Name}}{{.RequestParams}} {{.ResponseParams}} {
	m.{{.Name}}CalledTimes++
	m.{{.Name}}CalledWith = append(m.{{.Name}}CalledWith, {{ToInputInstance .RequestParams $mockName}})
	return m.Mock{{.Name}}({{ToParamNameList .RequestParams}})
}

{{ end}}