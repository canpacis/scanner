package structd

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Getter interface {
	Get(string) any
}

type caster interface {
	Cast(any, reflect.Type) (any, error)
}

type Unmarshaler interface {
	UnmarshalString(v string) error
}

type Decoder struct {
	getter Getter
	key    string
}

func (d *Decoder) Decode(v any) error {
	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return &InvalidUnmarshalError{rt}
	}
	rv = rv.Elem()
	rt = rt.Elem()
	if rv.Kind() != reflect.Struct {
		return &InvalidUnmarshalError{rt}
	}

	for i := range rv.NumField() {
		field := rt.Field(i)
		value := rv.Field(i)

		if !field.IsExported() {
			continue
		}

		tag, ok := field.Tag.Lookup(d.key)
		if !ok {
			continue
		}

		target := d.getter.Get(tag)
		if target == nil {
			continue
		}

		tv := reflect.ValueOf(target)
		tt := reflect.TypeOf(target)
		if tv.IsZero() {
			continue
		}

		if !tt.AssignableTo(field.Type) {
			c, ok := d.getter.(caster)
			if ok {
				casted, err := c.Cast(target, field.Type)
				if err != nil {
					return wrapCastErr(err)
				}
				value.Set(reflect.ValueOf(casted))
				continue
			}

			return &UnmarshalTypeError{
				Value:  tt.Name(),
				Type:   field.Type,
				Struct: rt.Name(),
				Field:  field.Name,
			}
		} else {
			value.Set(tv)
		}
	}

	return nil
}

type numbers interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

func parse[T numbers](s string) (T, error) {
	var n T
	rt := reflect.TypeOf(n)

	switch rt.Kind() {
	case reflect.Float32, reflect.Float64:
		i, err := strconv.ParseFloat(s, rt.Bits())
		if err != nil {
			return n, err
		}
		n = T(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(s, 10, rt.Bits())
		if err != nil {
			return n, err
		}
		n = T(i)
	default:
		i, err := strconv.ParseInt(s, 10, rt.Bits())
		if err != nil {
			return n, err
		}
		n = T(i)
	}

	return n, nil
}

const DefaultSeperator = ","

func DefaultCast(from any, to reflect.Type) (any, error) {
	switch from := from.(type) {
	case string:
		switch to.Kind() {
		case reflect.Uint8:
			return parse[uint8](from)
		case reflect.Uint16:
			return parse[uint16](from)
		case reflect.Uint32:
			return parse[uint32](from)
		case reflect.Uint64:
			return parse[uint64](from)
		case reflect.Int8:
			return parse[int8](from)
		case reflect.Int16:
			return parse[int16](from)
		case reflect.Int32:
			return parse[int32](from)
		case reflect.Int64:
			return parse[int64](from)
		case reflect.Uint:
			return parse[uint](from)
		case reflect.Int:
			return parse[int](from)
		case reflect.Float32:
			return parse[float32](from)
		case reflect.Float64:
			return parse[float64](from)
		case reflect.Bool:
			b, err := strconv.ParseBool(from)
			return b, err
		case reflect.Slice:
			split := strings.Split(from, DefaultSeperator)

			switch to.Elem().Kind() {
			case reflect.String:
				return split, nil
			default:
				result := reflect.New(to).Elem()

				for _, entry := range split {
					value, err := DefaultCast(entry, to.Elem())
					if err != nil {
						return nil, err
					}
					result = reflect.Append(result, reflect.ValueOf(value))
				}

				return result.Interface(), nil
			}
		default:
			toPtr := reflect.New(to)
			u, ok := toPtr.Interface().(Unmarshaler)
			if !ok {
				return nil, errors.ErrUnsupported
			}

			if err := u.UnmarshalString(from); err != nil {
				return nil, &UnmarshalerError{
					Err:         err,
					Value:       from,
					Unmarshaler: to,
				}
			}

			return toPtr.Elem().Interface(), nil
		}
	case uint, int, uint8, uint16, uint32, uint64, int8, int16, int32, int64, float32, float64:
		switch to.Kind() {
		case reflect.String:
			return fmt.Sprintf("%d", from), nil
		case reflect.Bool:
			return from != 0, nil
		default:
			return nil, errors.ErrUnsupported
		}
	case bool:
		var str = "0"
		if from {
			str = "1"
		}

		switch to.Kind() {
		case reflect.String:
			if from {
				return "true", nil
			}
			return "false", nil
		case reflect.Uint8:
			return parse[uint8](str)
		case reflect.Uint16:
			return parse[uint16](str)
		case reflect.Uint32:
			return parse[uint32](str)
		case reflect.Uint64:
			return parse[uint64](str)
		case reflect.Int8:
			return parse[int8](str)
		case reflect.Int16:
			return parse[int16](str)
		case reflect.Int32:
			return parse[int32](str)
		case reflect.Int64:
			return parse[int64](str)
		case reflect.Uint:
			return parse[uint](str)
		case reflect.Int:
			return parse[int](str)
		case reflect.Float32:
			return parse[float32](str)
		case reflect.Float64:
			return parse[float64](str)
		default:
			return nil, errors.ErrUnsupported
		}
	default:
		return nil, errors.ErrUnsupported
	}
}

func New(getter Getter, key string) *Decoder {
	return &Decoder{
		getter: getter,
		key:    key,
	}
}
