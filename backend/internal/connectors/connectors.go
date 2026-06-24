package connectors

import (
	"strings"

	"dw0rdwk/backend/internal/models"
)

const (
	KindGeneric = "generic"
	Kind29WK    = "29wk"
)

type OrderResponse struct {
	RemoteOrderID string
	Status        string
	Progress      string
	Remarks       string
}

type CourseQueryInput struct {
	Class    models.CourseClass
	School   string
	Account  string
	Password string
	Type     string
}

type CourseCandidate struct {
	ID   string
	Name string
	Raw  map[string]any
}

func NormalizeKind(kind string) string {
	normalized := strings.ToLower(strings.TrimSpace(kind))
	switch normalized {
	case "":
		return KindGeneric
	case "29", "29wk", "29网课", "29wangke", "29通用", "29_common", "29-common":
		return Kind29WK
	case "common", "常见货源", "common_source", "common-source":
		return KindGeneric
	case "custom", "自定义接口", "custom_api", "custom-api":
		return KindGeneric
	default:
		return strings.TrimSpace(kind)
	}
}

func Is29WKKind(kind string) bool {
	return NormalizeKind(kind) == Kind29WK
}
