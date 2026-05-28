package utils

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"ledger-backend/internal/config"
	"strconv"
)

type PaginationResult[T any] struct {
	Total int64 `json:"total"`
	List  []T   `json:"list"`
}

func Paginate(c *gin.Context) func(db *gorm.DB) *gorm.DB {
	defaultPageSize := strconv.Itoa(config.AppConfig.Server.Page.DefaultPageSize)
	defaultPageNum := strconv.Itoa(config.AppConfig.Server.Page.DefaultPageNum)
	maxPageSize := config.AppConfig.Server.Page.MaxPageSize

	return func(db *gorm.DB) *gorm.DB {
		page, _ := strconv.Atoi(c.DefaultQuery("page", defaultPageSize))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", defaultPageNum))

		if page <= 0 {
			page = 1
		}

		if pageSize <= 0 {
			pageSize = 10
		}

		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}

		return db.Offset((page - 1) * pageSize).Limit(pageSize)
	}

}
