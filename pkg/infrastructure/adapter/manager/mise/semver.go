package mise

import (
	"strconv"
	"strings"
)

func validIdentifiers(value string) bool {
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" || (len(identifier) > 1 && identifier[0] == '0' && allDigits(identifier)) {
			return false
		}
		for _, char := range identifier {
			if !isIdentifierChar(char) {
				return false
			}
		}
	}
	return true
}

func isIdentifierChar(char rune) bool {
	return char >= '0' && char <= '9' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char == '-'
}

func allDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func comparePrerelease(left, right string) int {
	if left == right {
		return 0
	}
	if left == "" {
		return 1
	}
	if right == "" {
		return -1
	}
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	for i := 0; i < len(leftParts) && i < len(rightParts); i++ {
		if compared := compareIdentifier(leftParts[i], rightParts[i]); compared != 0 {
			return compared
		}
	}
	if len(leftParts) < len(rightParts) {
		return -1
	}
	return 1
}

func compareIdentifier(left, right string) int {
	leftNumber, leftErr := strconv.Atoi(left)
	rightNumber, rightErr := strconv.Atoi(right)
	if leftErr == nil && rightErr == nil {
		if leftNumber < rightNumber {
			return -1
		}
		if leftNumber > rightNumber {
			return 1
		}
		return 0
	}
	if leftErr == nil {
		return -1
	}
	if rightErr == nil {
		return 1
	}
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
