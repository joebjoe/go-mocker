package main

import (
	"github.com/joebjoe/go-mocker/internal/mocker"
)

func main() {
	req := mocker.Request{
		Module:  "github.com/aws/aws-sdk-go-v2/service/sqs",
		Package: "sqs",
		Type:    "Client",
	}

	methods, err := mocker.FindMethods(req)

	if err != nil {
		panic(err)
	}

	req.Methods = methods

	if err := mocker.Generate(req); err != nil {
		panic(err)
	}
}
