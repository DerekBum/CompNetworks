package sum

import "encoding/binary"

func getSum(input []byte) uint16 {
	var res uint16
	if len(input)%2 != 0 {
		input = append([]byte{0}, input...)
	}
	for i := 0; i < len(input); i += 2 {
		res += binary.BigEndian.Uint16(input[i : i+2])
	}
	return ^uint16(0) - res
}

func validate(input []byte, sum uint16) bool {
	correct := getSum(input)
	return correct == sum
}
