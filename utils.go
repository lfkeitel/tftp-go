package main

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func totalStringMapLen(m map[string]string) int {
	size := 0

	for k, v := range m {
		size += len(k)
		size += len(v)
		size += 2 // NULL separators
	}

	return size
}

func parseOptions(options [][]byte) (*tftpOptions, map[string]string) {
	// Make copy of default options to adjust here
	base := defaultOptions.copy()

	optionLen := len(options)
	if optionLen < 2 {
		return base, nil
	}

	ackedOptions := make(map[string]string)

	if optionLen&1 == 1 { // Check option slice is even length
		options = options[:len(options)-1]
	}

	for i := 0; i < len(options); i += 2 {
		option := strings.ToLower(string(options[i])) // options names are case insensitive
		value := string(options[i+1])

		switch option {
		case optionBlockSize:
			val, err := strconv.Atoi(value)
			if err != nil {
				continue
			}

			if val < 8 || val > 65464 { // Request value out of range
				// Respond with default
				ackedOptions[optionBlockSize] = strconv.Itoa(base.blockSize)
				continue
			}
			base.blockSize = val
			ackedOptions[optionBlockSize] = value
		case optionTimeout:
			val, err := strconv.Atoi(value)
			if err != nil {
				continue
			}

			if val < 1 || val > 255 { // Request value out of range
				// Response with default
				ackedOptions[optionTimeout] = strconv.FormatInt(base.timeout.Nanoseconds()/int64(time.Second), 10)
				continue
			}
			base.timeout = time.Duration(val) * time.Second
			ackedOptions[optionTimeout] = value
		case optionTransferSize:
			val, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				continue
			}
			// The calling function is responsible for fulfilling tsize
			base.tsize = val
		}
	}

	return base, ackedOptions
}

func decodeUInt16(op []byte) uint16 {
	var code uint16

	switch len(op) {
	case 1:
		code = uint16(op[0])
	case 2:
		code = uint16(op[0]) << 8
		code = code + uint16(op[1])
	}

	return code
}

func encodeUInt16(in uint16) []byte {
	out := make([]byte, 2)
	out[0] = byte(in >> 8)
	out[1] = byte(in)
	return out
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
