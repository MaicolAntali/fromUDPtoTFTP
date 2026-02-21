package tftp

import (
	"reflect"
	"testing"
)

func TestParseRRQ(t *testing.T) {
	tests := []struct {
		name    string
		arg     []byte
		want    *RRQ
		wantErr bool
	}{
		{
			name:    "Standard RRQ format",
			arg:     []byte("\x00\x01test.txt\x00octet\x00"),
			want:    &RRQ{Filename: "test.txt", Mode: "octet"},
			wantErr: false,
		},
		{
			name:    "Malformed Packet: too short.",
			arg:     []byte("\x00\x01"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Malformed Packet: missing filename",
			arg:     []byte("\x00\x01someRandomText"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Malformed Packet: missing mode",
			arg:     []byte("\x00\x01test.txt\x00SomeRandomText"),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRRQ(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRRQ() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRRQ() got = %v, want %v", got, tt.want)
			}
		})
	}
}
