package pagination

// DefaultPageSize is the number of items per page when none is specified.
const DefaultPageSize = 20

// Page holds computed pagination state for a single page of results.
type Page struct {
	CurrentPage int
	PageSize    int
	TotalItems  int64
	TotalPages  int
	HasPrev     bool
	HasNext     bool
	PrevPage    int
	NextPage    int
	Offset      int
}

// New computes pagination state for the given page number, page size, and total item count.
// page is clamped to [1, TotalPages]. An empty result set always returns page 1 of 1.
func New(page, pageSize int, total int64) Page {
	if pageSize < 1 {
		pageSize = 1
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}

	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	return Page{
		CurrentPage: page,
		PageSize:    pageSize,
		TotalItems:  total,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
		Offset:      (page - 1) * pageSize,
	}
}
