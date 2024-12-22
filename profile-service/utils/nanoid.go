package utils

import (
	"crypto/rand"
	"math"
)

const (
	alphabet    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	defaultSize = 21
)

func getMask(alphabetSize int) int {
	for i := 1; i <= 8; i++ {
		mask := (2 << uint(i)) - 1
		if mask >= alphabetSize-1 {
			return mask
		}
	}
	return 0
}

func GenerateNanoId(l ...int) string {
	var size int = 0
	switch {
	case len(l) == 0:
		size = defaultSize
	case len(l) == 1:
		size = l[0]
		if size < 0 {
			size = defaultSize
		}
	}

	chars := []rune(alphabet)
	mask := getMask(len(chars))
	ceilArg := 1.6 * float64(mask*size) / float64(len(alphabet))
	step := int(math.Ceil(ceilArg))
	id := make([]rune, size)
	bytes := make([]byte, step)
	for j := 0; ; {
		_, err := rand.Read(bytes)
		if err != nil {
			return generateRandomString(size)
		}
		for i := 0; i < step; i++ {
			currByte := bytes[i] & byte(mask)
			if currByte < byte(len(chars)) {
				id[j] = chars[currByte]
				j++
				if j == size {
					return string(id[:size])
				}
			}
		}
	}
}
