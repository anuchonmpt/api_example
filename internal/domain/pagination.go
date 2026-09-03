package domain

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

func NormalizePagination(page, pageSize int) Pagination {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return Pagination{Page: page, PageSize: pageSize}
}

func (p Pagination) Limit() int32  { return int32(p.PageSize) }
func (p Pagination) Offset() int32 { return int32((p.Page - 1) * p.PageSize) }

type Page[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
}
