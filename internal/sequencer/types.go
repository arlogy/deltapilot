package sequencer

type ValueGenerator[T any] interface {
	Next() (T, error)
}

type ValueGetter[T any] interface {
	GetValue() (T, error)
}

type ValueSetter[T any] interface {
	SetValue(v T) error
}

// ValueAdvancer is a sequencer that advances its current value to the next generated value.
type ValueAdvancer[T any] interface {
	ValueGenerator[T]
	ValueGetter[T]
	ValueSetter[T]
}
