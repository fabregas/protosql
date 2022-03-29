package protosql

import "fmt"

const (
	defaultPageSize = 12
	maxPageSize     = 10000
)

type Pager interface {
	GetPageSize() uint32
	GetCurrentPage() uint32
}

func correctingPageSize(s uint32) uint32 {
	if s == 0 {
		s = defaultPageSize
	}
	if s > maxPageSize {
		s = maxPageSize
	}

	return s
}

func pageQuery(p Pager) string {
	pageSize := correctingPageSize(p.GetPageSize())
	query := fmt.Sprintf(" LIMIT %d", pageSize)
	if n := p.GetCurrentPage(); n > 0 {
		query += fmt.Sprintf(" OFFSET %d", n*pageSize)
	}

	return query
}

type page struct {
	size uint32
	num  uint32
}

func (p page) GetPageSize() uint32 {
	return p.size
}

func (p page) GetCurrentPage() uint32 {
	return p.num
}

func Page(num, size uint32) page {
	return page{size: size, num: num}
}
