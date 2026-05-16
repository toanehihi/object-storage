package response

type PaginationResponse[T any] struct {
	Data       []T      `json:"data"`
	Pagination PageMeta `json:"pagination"`
}

type PageMeta struct {
	Page       int32  `json:"page"`
	PageSize   int32  `json:"pageSize"`
	Total      int64  `json:"total"`
	TotalPages int32  `json:"totalPages"`
	NextPage   *int32 `json:"nextPage,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
}

func Paginated[T any](data []T, page, pageSize int32, total int64) PaginationResponse[T] {
	totalPages := int32((total + int64(pageSize) - 1) / int64(pageSize))

	var nextPage *int32
	if page < totalPages {
		np := page + 1
		nextPage = &np
	}

	if data == nil {
		data = []T{}
	}

	return PaginationResponse[T]{
		Data: data,
		Pagination: PageMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
			NextPage:   nextPage,
		},
	}
}