package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPaginationPage  = 1
	DefaultPaginationLimit = 10
	MaxPaginationLimit     = 100
)

func parsePaginationParams(c *gin.Context) (page, limit int) {
	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	if pageStr == "" && limitStr == "" {
		return DefaultPaginationPage, 0
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = DefaultPaginationPage
	}

	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = DefaultPaginationLimit
	}
	if limit > MaxPaginationLimit {
		limit = MaxPaginationLimit
	}

	return page, limit
}
