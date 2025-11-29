package utils

type SetVal interface {
	string | int | int16 | int32 | int8 | int64
}

type Set[T SetVal] map[T]bool

func (s Set[T]) Add(item T) {
	s[item] = true
}

func (s Set[T]) Remove(item T) {
	delete(s, item)
}

func (s Set[T]) Has(item T) bool {
	_, exist := s[item]
	return exist
}

func (s Set[T]) ToList() []T {
	keys := make([]T, 0, len(s))
	for key := range s {
		keys = append(keys, key)
	}
	return keys
}
