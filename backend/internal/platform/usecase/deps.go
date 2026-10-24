package usecase

import "reflect"

// AnyNil reports whether one of deps is missing: a nil interface, or a nil
// pointer, func, map, slice or channel. A NewService checks its required
// dependencies with it.
func AnyNil(deps ...any) bool {
	for _, d := range deps {
		if d == nil {
			return true
		}
		switch v := reflect.ValueOf(d); v.Kind() {
		case reflect.Pointer, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan, reflect.Interface:
			if v.IsNil() {
				return true
			}
		default:
		}
	}
	return false
}
