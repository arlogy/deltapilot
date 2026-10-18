package sequencer

type Generator[T any] interface {
	Next() (T, error)
}

type ValueProvider[T any] interface {
	GetCurrent() (T, error)
}

type ValueSetter[T any] interface {
	SetCurrent(v T) error
}

// ValueAdvancer generates a new value based on the current value.
type ValueAdvancer[T any] interface {
	Generator[T]
	ValueProvider[T]
	ValueSetter[T]
}
