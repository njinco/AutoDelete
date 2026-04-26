package autodelete

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSlashSetArgs(t *testing.T) {
	args, err := slashSetArgs([]slashInteractionOption{
		{Name: "duration", Value: "24h"},
		{Name: "count", Value: float64(100)},
	})
	if err != nil {
		t.Fatalf("slashSetArgs returned error: %v", err)
	}

	want := []string{"24h", "100"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("slashSetArgs = %#v, want %#v", args, want)
	}
}

func TestSlashSetArgsRejectsBadDuration(t *testing.T) {
	_, err := slashSetArgs([]slashInteractionOption{
		{Name: "duration", Value: float64(24)},
	})
	if err == nil {
		t.Fatal("slashSetArgs accepted a non-string duration")
	}
}

func TestSlashIntegerValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  int64
	}{
		{name: "float64", value: float64(42), want: 42},
		{name: "int64", value: int64(43), want: 43},
		{name: "int", value: int(44), want: 44},
		{name: "jsonNumber", value: json.Number("45"), want: 45},
		{name: "string", value: "46", want: 46},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := slashIntegerValue(tt.value)
			if err != nil {
				t.Fatalf("slashIntegerValue returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("slashIntegerValue = %d, want %d", got, tt.want)
			}
		})
	}
}
