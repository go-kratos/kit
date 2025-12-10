package pagination

// PageRequest defines the interface for requests that contain pagination parameters.
type PageRequest interface {
	GetPageNum() int32
	GetPageSize() int32
}

// Range holds calculated offset and limit values.
type Range struct {
	Offset int32
	Limit  int32
}

// Paginator defines the interface for resolving pagination parameters.
type Paginator interface {
	// Resolve calculates the offset and limit based on the provided page and size.
	Resolve(page, size int32) Range
	Parse(req PageRequest) Range
}

// NewPaginator creates a new Pagination instance with default page and size.
func NewPaginator(defaultPage, defaultSize int32) Paginator {
	return &paginator{
		Page: defaultPage,
		Size: defaultSize,
	}
}

// paginator holds default paginator settings.
type paginator struct {
	Page int32
	Size int32
}

// Resolve calculates the offset and limit based on the provided page and size,
// applying defaults when page/size are <= 0.
func (p *paginator) Resolve(page, size int32) Range {
	if page <= 0 {
		page = p.Page
	}
	if size <= 0 {
		size = p.Size
	}
	offset := (page - 1) * size
	return Range{
		Offset: offset,
		Limit:  size,
	}
}

// Parse extracts pagination parameters from a PageRequest and resolves them.
func (p *paginator) Parse(req PageRequest) Range {
	return p.Resolve(req.GetPageNum(), req.GetPageSize())
}
