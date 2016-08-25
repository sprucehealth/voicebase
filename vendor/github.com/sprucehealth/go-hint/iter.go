package hint

// ListMeta is the structure that contains the common properties
// of List iterators.
type ListMeta struct {
	CurrentCount uint64 `json:"count"`
	TotalCount   uint64 `json:"total_count"`
}

// Query is the function used to get a page listing.
type Query func(params *ListParams) ([]interface{}, ListMeta, error)

// Iter is a structure used for generic pagination through a list of resources.
// NextPage is retrieved by calling the Query function.
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

// GetIter returns an implementation of an iterator based on the
// params and the query function.
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

// Next returns the next item in the list of resources, querying the source
// for the next page if there is more to query. It returns false when done querying
// or the query results in an error.
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

// Current returns the current item the iterator points to.
func (it *Iter) Current() interface{} {
	return it.cur
}

// Err returns an error the iterator is holding on to as a result
// of querying against list of resources.
func (it *Iter) Err() error {
	return it.err
}
