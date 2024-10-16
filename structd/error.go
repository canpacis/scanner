package structd

import (
	"reflect"
)

type CastError struct {
	Err error
}

func (e *CastError) Error() string {
	if e.Err == nil {
		return ""
	}

	return "cast error: " + e.Err.Error()
}

func (e *CastError) Unwrap() error {
	return e.Err
}

func wrapCastErr(err error) error {
	if err == nil {
		return nil
	}

	return &CastError{
		Err: err,
	}
}

type UnmarshalerError struct {
	Err         error
	Value       string
	Unmarshaler reflect.Type
}

func (e *UnmarshalerError) Error() string {
	if e.Err == nil {
		return ""
	}

	return "structd: unmarshaler " + e.Unmarshaler.Name() + " failed to unmarshal value '" + e.Value + "': " + e.Err.Error()
}

func (e *UnmarshalerError) Unwrap() error {
	return e.Err
}

// An InvalidUnmarshalError describes an invalid argument passed to [Unmarshal].
// (The argument to [Unmarshal] must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "structd: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Pointer {
		return "structd: Unmarshal(non-pointer " + e.Type.String() + ")"
	}

	if e.Type.Elem().Kind() != reflect.Struct {
		return "structd: Unmarshal(non-struct " + e.Type.String() + ")"
	}

	return "structd: Unmarshal(nil " + e.Type.String() + ")"
}

// An UnmarshalTypeError describes a value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value  string       // description of a value - "bool", "array", "number -5"
	Type   reflect.Type // type of Go value it could not be assigned to
	Struct string       // name of the struct type containing the field
	Field  string       // the full path from root node to the field, include embedded struct
}

func (e *UnmarshalTypeError) Error() string {
	if e.Struct != "" || e.Field != "" {
		return "structd: cannot unmarshal " + e.Value + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.Type.String()
	}
	return "structd: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
}
