package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"ledger-backend/internal/config"
)

func page[T any](c *gin.Context, q *gorm.DB, out *[]T) {
	pageNum, pageSize := paginationParams(c)
	var total int64
	q.Count(&total)
	q.Offset((pageNum - 1) * pageSize).Limit(pageSize).Find(out)
	ok(c, gin.H{"total": total, "list": out, "page": pageNum, "page_size": pageSize})
}

func paginationParams(c *gin.Context) (int, int) {
	pageNum, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(config.AppConfig.Server.Page.DefaultPageNum)))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(config.AppConfig.Server.Page.DefaultPageSize)))
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > config.AppConfig.Server.Page.MaxPageSize {
		pageSize = config.AppConfig.Server.Page.MaxPageSize
	}
	return pageNum, pageSize
}
