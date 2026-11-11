package reactive

import "reflect"

type Option[T any] func(*config[T])

type config[T any] struct {
	eq func(a, b T) bool
}

func WithEqual[T any](eq func(a, b T) bool) Option[T] {
	return func(c *config[T]) { c.eq = eq }
}

func resolveEqual[T any](opts []Option[T]) func(a, b T) bool {
	var c config[T]
	for _, o := range opts {
		o(&c)
	}
	if c.eq != nil {
		return c.eq
	}
	return defaultEqual[T]()
}

func basicEqual[T any]() func(a, b T) bool {
	var zero T
	var f any
	switch any(zero).(type) {
	case string:
		f = func(a, b string) bool { return a == b }
	case bool:
		f = func(a, b bool) bool { return a == b }
	case int:
		f = func(a, b int) bool { return a == b }
	case int8:
		f = func(a, b int8) bool { return a == b }
	case int16:
		f = func(a, b int16) bool { return a == b }
	case int32:
		f = func(a, b int32) bool { return a == b }
	case int64:
		f = func(a, b int64) bool { return a == b }
	case uint:
		f = func(a, b uint) bool { return a == b }
	case uint8:
		f = func(a, b uint8) bool { return a == b }
	case uint16:
		f = func(a, b uint16) bool { return a == b }
	case uint32:
		f = func(a, b uint32) bool { return a == b }
	case uint64:
		f = func(a, b uint64) bool { return a == b }
	case uintptr:
		f = func(a, b uintptr) bool { return a == b }
	case float32:
		f = func(a, b float32) bool { return a == b }
	case float64:
		f = func(a, b float64) bool { return a == b }
	default:
		return nil
	}
	eq, _ := f.(func(a, b T) bool)
	return eq
}

func defaultEqual[T any]() func(a, b T) bool {
	if eq := basicEqual[T](); eq != nil {
		return eq
	}
	t := reflect.TypeFor[T]()
	switch {
	case staticallyComparable(t):
		return func(a, b T) bool { return any(a) == any(b) }
	case mayBeComparable(t):
		return func(a, b T) bool {
			va, vb := reflect.ValueOf(&a).Elem(), reflect.ValueOf(&b).Elem()
			return va.Comparable() && vb.Comparable() && va.Equal(vb)
		}
	}
	return func(T, T) bool { return false }
}

func staticallyComparable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface, reflect.Func, reflect.Map, reflect.Slice:
		return false
	case reflect.Array:
		return staticallyComparable(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !staticallyComparable(t.Field(i).Type) {
				return false
			}
		}
	}
	return true
}

func mayBeComparable(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return false
	case reflect.Array:
		return mayBeComparable(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if !mayBeComparable(t.Field(i).Type) {
				return false
			}
		}
	}
	return true
}
