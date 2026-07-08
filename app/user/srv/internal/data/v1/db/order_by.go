package db

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var userOrderColumns = map[string]string{
	"id":          "id",
	"username":    "username",
	"mobile":      "mobile",
	"email":       "email",
	"nick_name":   "nick_name",
	"birthday":    "birthday",
	"gender":      "gender",
	"role":        "role",
	"add_time":    "add_time",
	"update_time": "update_time",
}

func applyOrderBy(query *gorm.DB, orderBy []string, allowedColumns map[string]string) *gorm.DB {
	for _, value := range orderBy {
		column, desc, ok := parseOrderBy(value, allowedColumns)
		if !ok {
			continue
		}
		query = query.Order(clause.OrderByColumn{
			Column: clause.Column{Name: column},
			Desc:   desc,
		})
	}
	return query
}

func parseOrderBy(value string, allowedColumns map[string]string) (string, bool, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "`", ""))
	if value == "" {
		return "", false, false
	}

	parts := strings.Fields(value)
	if len(parts) == 0 || len(parts) > 2 {
		return "", false, false
	}

	column, ok := allowedColumns[strings.ToLower(parts[0])]
	if !ok {
		return "", false, false
	}

	if len(parts) == 1 {
		return column, false, true
	}

	switch strings.ToLower(parts[1]) {
	case "asc":
		return column, false, true
	case "desc":
		return column, true, true
	default:
		return "", false, false
	}
}
