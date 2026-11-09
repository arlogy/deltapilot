package sequencer

// ValueGenerator is a sequencer that generates values. Uniqueness of generated values depends on the
// implementation.
type ValueGenerator[T any] interface {
	Next() (T, error)
}

// ValueAdvancer is a ValueGenerator that generates the next value from its current value.
type ValueAdvancer[T any] interface {
	ValueGenerator[T]
	GetValue() (T, error)
	SetValue(v T) error
}
