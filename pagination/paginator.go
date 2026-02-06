package pagination

// PageRequest defines the interface for requests that contain pagination parameters.
type PageRequest interface {
	GetPageNum() int32
	GetPageSize() int32
}

// PageRange holds calculated offset and limit values.
type PageRange struct {
	Offset int32
	Limit  int32
}

// Paginator defines the interface for resolving pagination parameters.
type Paginator interface {
	// Resolve calculates the offset and limit based on the provided page and size.
	Resolve(page, size int32) PageRange
	Parse(req PageRequest) PageRange
}

// NewPaginator creates a new Pagination instance with default size and optional max page size.
func NewPaginator(defaultPageSize, maxPageSize int32) Paginator {
	return &paginator{
		DefaultPageSize: defaultPageSize,
		MaxPageSize:     maxPageSize,
	}
}

// paginator holds default paginator settings.
type paginator struct {
	DefaultPageSize int32
	MaxPageSize     int32
}

// Resolve calculates the offset and limit based on the provided page and size,
// applying defaults when page/size are <= 0.
func (p *paginator) Resolve(page, size int32) PageRange {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = p.DefaultPageSize
	}
	if p.MaxPageSize > 0 && size > p.MaxPageSize {
		size = p.MaxPageSize
	}
	offset := (page - 1) * size
	return PageRange{
		Offset: offset,
		Limit:  size,
	}
}

// Parse extracts pagination parameters from a PageRequest and resolves them.
func (p *paginator) Parse(req PageRequest) PageRange {
	return p.Resolve(req.GetPageNum(), req.GetPageSize())
}
