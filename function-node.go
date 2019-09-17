package main

import (
	"strings"
)

func ConvertHearderKey(header string) string {
	header = strings.ToLower(header)
	headerArray := strings.Split(header, "-")
	h := ""
	arrLen := len(headerArray)
	for i, value := range headerArray {
		vLen := len(value)
		if vLen < 1 {
			continue
		} else {
			if vLen == 1 {
				h += strings.ToUpper(value)
			} else {
				h += strings.ToUpper(string(value[0])) + value[1:]
			}
			if i != arrLen-1 {
				h += "-"
			}
		}
	}
	return h
}
