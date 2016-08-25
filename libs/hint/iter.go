package hint

// ListMeta is the structure that contains the common properties
// of List iterators.
type ListMeta struct {
	CurrentCount uint64 `json:"count"`
	TotalCount   uint64 `json:"total_count"`
}

// Query is the function used to get a page listing.
type Query func(params *ListParams) ([]interface{}, ListMeta, error)

type Iter struct {
	query        Query
	err          error
	cur          interface{}
	values       []interface{}
	totalQueried uint64
	hasMore      bool
	params       *ListParams
	meta         ListMeta
}

func GetIter(params *ListParams, query Query) *Iter {
	iter := &Iter{
		params: params,
	}
	iter.query = query
	iter.getPage()
	return iter
}

func (it *Iter) getPage() error {
	it.values, it.meta, it.err = it.query(it.params)
	it.totalQueried += uint64(len(it.values))
	it.hasMore = it.totalQueried < it.meta.TotalCount
	return it.err
}

func (it *Iter) Next() bool {
	if len(it.values) == 0 && it.hasMore {
		it.params.Offset = it.totalQueried
		if err := it.getPage(); err != nil {
			return false
		}
	}
	if len(it.values) == 0 {
		return false
	}

	it.cur = it.values[0]
	it.values = it.values[1:]
	return true
}

func (it *Iter) Current() interface{} {
	return it.cur
}

func (it *Iter) Err() error {
	return it.err
}
