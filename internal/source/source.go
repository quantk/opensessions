package source

import "strings"

type Kind string

const (
	KindOpenCode Kind = "opencode"
	KindPi       Kind = "pi"
)

func NormalizeKind(kind string) Kind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case string(KindPi):
		return KindPi
	case string(KindOpenCode), "":
		return KindOpenCode
	default:
		return Kind(strings.ToLower(strings.TrimSpace(kind)))
	}
}

func KindString(kind string) string {
	return string(NormalizeKind(kind))
}

func NamespacedID(kind Kind, parts ...string) string {
	clean := make([]string, 0, len(parts)+1)
	clean = append(clean, string(NormalizeKind(string(kind))))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			clean = append(clean, part)
		}
	}
	return strings.Join(clean, ":")
}
