package util

import "unicode"

func CapitalizeFirstLetter(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	if message == "" {
		return ""
	}

	runes := []rune(message)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
