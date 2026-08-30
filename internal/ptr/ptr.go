package ptr

import "fmt"

func CloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func EqualString(str1 *string, str2 *string) bool {
	if str1 == nil || str2 == nil {
		return str1 == str2
	}
	return *str1 == *str2
}

func StringQuotedOrNull(value *string) string {
	if value == nil {
		return "<null>"
	}
	return fmt.Sprintf("%q", *value)
}
