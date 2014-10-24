package ratelimit

type Null struct{}

func (Null) Check(cost int) (bool, error) {
	return true, nil
}

type NullKeyed struct{}

func (NullKeyed) Check(key string, cost int) (bool, error) {
	return true, nil
}
