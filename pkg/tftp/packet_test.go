package tftp

import (
	"reflect"
	"testing"
)

func TestParseRWRQ(t *testing.T) {
	tests := []struct {
		name    string
		arg     []byte
		want    *RWRQ
		wantErr bool
	}{
		{
			name:    "Standard RWRQ format",
			arg:     []byte("\x00\x01test.txt\x00octet\x00"),
			want:    &RWRQ{Filename: "test.txt", Mode: "octet"},
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
			got, err := ParseRWRQ(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRWRQ() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseRWRQ() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMarshalDATA(t *testing.T) {
	tests := []struct {
		name    string
		fields  DATA
		want    []byte
		wantErr bool
	}{
		{
			name:    "Standard DATA packet",
			fields:  DATA{BlockId: 1, Payload: []byte("This is a DATA packet")},
			want:    []byte("\x00\x03\x00\x01This is a DATA packet"),
			wantErr: false,
		},
		{
			name:    "Malformed Packet",
			fields:  DATA{BlockId: 1, Payload: []byte("This is a DATA packet")},
			want:    []byte("\x00\x03\x00\x01This is a DATA packet"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pd := DATA{
				BlockId: tt.fields.BlockId,
				Payload: tt.fields.Payload,
			}
			got, err := pd.Marshal()
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Marshal() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDATA(t *testing.T) {
	tests := []struct {
		name    string
		arg     []byte
		want    *DATA
		wantErr bool
	}{
		{
			name:    "Standard DATA format",
			arg:     []byte("\x00\x01\x00\x05fooBar"),
			want:    &DATA{BlockId: 5, Payload: []byte("fooBar")},
			wantErr: false,
		},
		{
			name:    "Short DATA packet",
			arg:     []byte("\x00\x01"),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseData(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseData() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseACK(t *testing.T) {
	tests := []struct {
		name    string
		arg     []byte
		want    *ACK
		wantErr bool
	}{
		{
			name:    "Standard ACK format",
			arg:     []byte("\x00\x04\x00\x01"),
			want:    &ACK{BlockId: 1},
			wantErr: false,
		},
		{
			name:    "Malformed ACK: too short",
			arg:     []byte("\x00\x04"),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseACK(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseACK() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseACK() got = %v, want %v", got, tt.want)
			}
		})
	}
}
