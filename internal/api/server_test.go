package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_validateRequestGET(t *testing.T) {
	type args struct {
		req RequestGET
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "missing module",
			args: args{
				req: RequestGET{
					Module:  "",
					Package: "some package",
					Type:    "some type",
					From:    "struct",
					To:      "mock",
				},
			},
			want: "'module' cannot be empty",
		},
		{
			name: "missing package",
			args: args{
				req: RequestGET{
					Module:  "some module",
					Package: "",
					Type:    "some type",
					From:    "struct",
					To:      "mock",
				},
			},
			want: "'package' cannot be empty",
		},
		{
			name: "missing type",
			args: args{
				req: RequestGET{
					Module:  "some module",
					Package: "some package",
					Type:    "",
					From:    "struct",
					To:      "mock",
				},
			},
			want: "'type' cannot be empty",
		},
		{
			name: "invalid mapping",
			args: args{
				req: RequestGET{
					Module:  "some module",
					Package: "some package",
					Type:    "some type",
					From:    "struct",
					To:      "not mock",
				},
			},
			want: "invalid mapping 'struct/not mock'; must be one of 'struct/interface', " +
				"'struct/mock', or 'interface/mock'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if err := validateRequestGET(tt.args.req); err != nil {
				got = err.Error()
			}
			assert.Equalf(t, tt.want, got, "missing or unexpected error")
		})
	}
}
