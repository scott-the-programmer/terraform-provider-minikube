package lib

import (
	"testing"
)

func Test_machineLogBridge_Write(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		lb      machineLogBridge
		args    args
		wantN   int
		wantErr bool
	}{
		{
			name:    "VirtualizationException",
			lb:      machineLogBridge{},
			args:    args{b: []byte("VirtualizationException")},
			wantN:   len([]byte("VirtualizationException")),
			wantErr: false,
		},
		{
			name:    "Warning",
			lb:      machineLogBridge{},
			args:    args{b: []byte("(1)warning")},
			wantN:   len([]byte("(1)warning")),
			wantErr: false,
		},
		{
			name:    "Environment",
			lb:      machineLogBridge{},
			args:    args{b: []byte("&exec.Cmd")},
			wantN:   len([]byte("&exec.Cmd")),
			wantErr: false,
		},
		{
			name:    "Info",
			lb:      machineLogBridge{},
			args:    args{b: []byte("i am an info log")},
			wantN:   len([]byte("i am an info log")),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := machineLogBridge{}
			gotN, err := lb.Write(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("machineLogBridge.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("machineLogBridge.Write() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}

func Test_stdLogBridge_Write(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		lb      stdLogBridge
		args    args
		wantN   int
		wantErr bool
	}{
		{
			name:    "Good",
			lb:      stdLogBridge{},
			args:    args{b: []byte("abc.txt:53: this is a log")},
			wantN:   len([]byte("abc.txt:53: this is a log")),
			wantErr: false,
		},
		{
			name:    "Error",
			lb:      stdLogBridge{},
			args:    args{b: []byte("abc.txt:a: this is a bad log")},
			wantN:   len([]byte("abc.txt:a: this is a bad log")),
			wantErr: false,
		},
		{
			name:    "Invalid",
			lb:      stdLogBridge{},
			args:    args{b: []byte("asdjkasdhasd")},
			wantN:   0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lb := stdLogBridge{}
			gotN, err := lb.Write(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("stdLogBridge.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("stdLogBridge.Write() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}
