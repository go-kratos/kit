package pagination

// Range holds calculated offset and limit values.
type Range struct {
	Offset int32
	Limit  int32
}

// Pagination holds default pagination settings.
type Pagination struct {
	Page int32
	Size int32
}

// New creates a new Pagination instance with default page and size.
func New(defaultPage, defaultSize int32) *Pagination {
	return &Pagination{
		Page: defaultPage,
		Size: defaultSize,
	}
}

// Resolve calculates the offset and limit based on the provided page and size,
// applying defaults when page/size are <= 0.
func (p *Pagination) Resolve(page, size int32) Range {
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
