package user

import (
	"strings"
)

func redactMobileForLog(mobile string) string {
	mobile = strings.TrimSpace(mobile)
	if len(mobile) < 7 {
		return "***"
	}
	return mobile[:3] + "****" + mobile[len(mobile)-4:]
}

func redactEmailForLog(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}

	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain == "" {
		return "***"
	}
	if len(local) == 1 {
		return "*@" + domain
	}
	return local[:1] + "***@" + domain
}

func redactOptionalEmailForLog(email *string) string {
	if email == nil {
		return ""
	}
	return redactEmailForLog(*email)
}

func redactIdentifierForLog(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}
	if strings.Contains(identifier, "@") {
		return redactEmailForLog(identifier)
	}
	if len(identifier) == 11 {
		return redactMobileForLog(identifier)
	}
	if len(identifier) <= 2 {
		return "***"
	}
	return identifier[:1] + "***" + identifier[len(identifier)-1:]
}
