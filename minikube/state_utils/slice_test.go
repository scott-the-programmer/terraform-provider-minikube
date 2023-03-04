package state_utils

import (
	"reflect"
	"testing"
)

func TestSliceOrNil(t *testing.T) {
	type args struct {
		slice []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Should return non nil slice of type T",
			args: args{
				slice: []string{"abc"},
			},
			want: []string{"abc"},
		},
		{
			name: "Should return nil",
			args: args{
				slice: []string{},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SliceOrNil(tt.args.slice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SliceOrNil() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadSliceState(t *testing.T) {
	type args struct {
		slice interface{}
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Should return string slice",
			args: args{
				slice: []string{"abc"},
			},
			want: []string{"abc"},
		},
		{
			name: "Should construct string array",
			args: args{
				slice: []interface{}{"abc"},
			},
			want: []string{"abc"},
		},
		{
			name: "Should return empty string array on unexpected type",
			args: args{
				slice: []int{},
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadSliceState(tt.args.slice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadSliceState() = %v, want %v", got, tt.want)
			}
		})
	}
}
