package dsmr

import "fmt"

type unsupportedObjectError struct {
	obj string
}

func (e *unsupportedObjectError) Error() string {
	return fmt.Sprintf("unsupported number of values in dsmr object: %q", e.obj)
}
