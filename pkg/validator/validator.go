package validator

import (
	"fmt"
	"net/mail"
	"strconv"
	"strings"
)

func Email(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Address != normalized {
		return "", fmt.Errorf("invalid email address")
	}
	return normalized, nil
}

func Pagination(pageValue, pageSizeValue string, defaultSize, maxSize int) (page, pageSize int, err error) {
	page, pageSize = 1, defaultSize
	if pageValue != "" {
		page, err = strconv.Atoi(pageValue)
		if err != nil || page < 1 {
			return 0, 0, fmt.Errorf("page must be a positive integer")
		}
	}
	if pageSizeValue != "" {
		pageSize, err = strconv.Atoi(pageSizeValue)
		if err != nil || pageSize < 1 || pageSize > maxSize {
			return 0, 0, fmt.Errorf("page_size must be between 1 and %d", maxSize)
		}
	}
	return page, pageSize, nil
}
