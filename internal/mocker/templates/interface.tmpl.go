package {{IFacePackageName .Package .Type}}
{{$typeName := IFaceTypeName .Package .Type}}
// {{$typeName}} ...
type {{$typeName}} interface {
	{{- range .Methods}}
	{{.Name}}{{.RequestParams}} {{.ResponseParams}}
	{{- end}}	
}