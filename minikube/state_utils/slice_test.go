package state_utils

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
		{
			name: "Should return array given schema.Set",
			args: args{
				slice: schema.NewSet(schema.HashString, []interface{}{"addon1", "addon2"}),
			},
			want: []string{"addon1", "addon2"},
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

func TestSetToSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    *schema.Set
		expected []string
	}{
		{
			name:     "Empty set",
			input:    schema.NewSet(schema.HashString, []interface{}{}),
			expected: []string{},
		},
		{
			name:     "Set with single item",
			input:    schema.NewSet(schema.HashString, []interface{}{"apple"}),
			expected: []string{"apple"},
		},
		{
			name:     "Set with multiple items",
			input:    schema.NewSet(schema.HashString, []interface{}{"banana", "apple", "cherry"}),
			expected: []string{"apple", "banana", "cherry"},
		},
		{
			name:     "Set with duplicate items",
			input:    schema.NewSet(schema.HashString, []interface{}{"apple", "banana", "apple", "cherry"}),
			expected: []string{"apple", "banana", "cherry"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SetToSlice(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("SetToSlice() = %v, want %v", result, tt.expected)
			}
		})
	}
}
