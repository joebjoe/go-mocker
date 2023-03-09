package api

type ToType string

const (
	ToTypeMock      ToType = "mock"
	ToTypeInterface ToType = "interface"
)

type FromType string

const (
	FromTypeStruct    FromType = "struct"
	FromTypeInterface FromType = "interface"
)

type RequestGET struct {
	Module  string   `json:"module"`
	Package string   `json:"package"`
	Type    string   `json:"type"`
	To      ToType   `param:"to"`
	From    FromType `param:"from"`
}
