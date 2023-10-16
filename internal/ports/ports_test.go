package ports

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		def     string
		want    *Definition
		wantErr bool
	}{
		{
			"simple",
			"1-10:101-110",
			&Definition{
				SourceStart: 1,
				SourceEnd:   10,
				DestStart:   101,
				DestEnd:     110,
			},
			false,
		},
		{
			"bad target range",
			"1-10:100-90",
			nil,
			true,
		},
		{
			"bad source range",
			"10-1:100-110",
			nil,
			true,
		},
		{
			"bad range length",
			"1-10:100-110",
			nil,
			true,
		},
		{
			"single port",
			"1-1:100-100",
			&Definition{
				SourceStart: 1,
				SourceEnd:   1,
				DestStart:   100,
				DestEnd:     100,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.def)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() got = %v, want %v", got, tt.want)
			}
		})
	}
}
