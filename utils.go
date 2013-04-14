package prolog

import (
	"strings"
)

func appendIndent(s, indent string) string {
	return indent + strings.Replace(s, "\n", "\n"+indent, -1)
}
