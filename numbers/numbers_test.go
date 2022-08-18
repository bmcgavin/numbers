package numbers

import "testing"
import "reflect"

func TestChecksum(t *testing.T) {
	type test struct {
		input []uint32
		want  uint32
	}

	tests := []test{
		{input: []uint32{uint32(1327694389), uint32(648333537)}, want: uint32(639769904)},
	}

	for _, tc := range tests {

		got := Checksum(tc.input)
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}
