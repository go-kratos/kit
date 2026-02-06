package pagination

import "testing"

type pageRequest struct {
	page int32
	size int32
}

func (r pageRequest) GetPageNum() int32 {
	return r.page
}

func (r pageRequest) GetPageSize() int32 {
	return r.size
}

func TestResolveDefaultPageIsOne(t *testing.T) {
	p := NewPaginator(20, 100)

	got := p.Resolve(0, 5)
	if got.Offset != 0 || got.Limit != 5 {
		t.Fatalf("expected offset=0 limit=5, got offset=%d limit=%d", got.Offset, got.Limit)
	}
}

func TestResolveUsesDefaultSizeWhenNonPositive(t *testing.T) {
	p := NewPaginator(20, 100)

	got := p.Resolve(2, 0)
	if got.Offset != 20 || got.Limit != 20 {
		t.Fatalf("expected offset=20 limit=20, got offset=%d limit=%d", got.Offset, got.Limit)
	}
}

func TestResolveAppliesMaxPageSize(t *testing.T) {
	p := NewPaginator(20, 50)

	got := p.Resolve(2, 100)
	if got.Offset != 50 || got.Limit != 50 {
		t.Fatalf("expected offset=50 limit=50, got offset=%d limit=%d", got.Offset, got.Limit)
	}
}

func TestParseUsesResolveRules(t *testing.T) {
	p := NewPaginator(20, 30)

	got := p.Parse(pageRequest{page: 0, size: 100})
	if got.Offset != 0 || got.Limit != 30 {
		t.Fatalf("expected offset=0 limit=30, got offset=%d limit=%d", got.Offset, got.Limit)
	}
}
