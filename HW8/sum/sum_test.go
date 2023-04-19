package sum

import "testing"

var arraysToSum = []struct {
	name string
	in   []byte
	out  uint16
}{
	{"full sum", []byte{255, 255}, uint16(0)},
	{"odd length", []byte{255}, uint16(65280)},
	{"big array", []byte{4, 23, 102, 244, 50, 2}, uint16(25330)},
	{"zero sum", []byte{0}, uint16(65535)},
	{"overflow", []byte{255, 255, 255, 255}, uint16(1)},
}

func TestSum(t *testing.T) {
	for _, tt := range arraysToSum {
		t.Run(tt.name, func(t *testing.T) {
			got := getSum(tt.in)
			if got != tt.out {
				t.Errorf("got %d, want %d", got, tt.out)
			}
		})
	}
}

var arraysToValidate = []struct {
	name string
	in   []byte
	sum  uint16
	out  bool
}{
	{"correct sum", []byte{255, 255}, uint16(0), true},
	{"incorrect sum", []byte{255}, uint16(65281), false},
	{"overflow correct", []byte{255, 255, 255, 255}, uint16(1), true},
	{"overflow period", []byte{255, 255, 255, 255}, uint16(256), false},
}

func TestValidate(t *testing.T) {
	for _, tt := range arraysToValidate {
		t.Run(tt.name, func(t *testing.T) {
			got := validate(tt.in, tt.sum)
			if got != tt.out {
				t.Errorf("got %v, want %v", got, tt.out)
			}
		})
	}
}
