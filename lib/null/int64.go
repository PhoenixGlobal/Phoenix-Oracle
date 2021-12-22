package null

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

type Int64 struct {
	Int64 int64
	Valid bool
}

func NewInt64(i int64, valid bool) Int64 {
	return Int64{
		Int64: i,
		Valid: valid,
	}
}

func Int64From(i int64) Int64 {
	return NewInt64(i, true)
}

func (i *Int64) UnmarshalJSON(data []byte) error {
	var err error
	var v interface{}
	if err = json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case float64:
		err = json.Unmarshal(data, &i.Int64)
	case string:
		str := string(x)
		if len(str) == 0 {
			i.Valid = false
			return nil
		}
		i.Int64, err = parse64(str)
	case nil:
		i.Valid = false
		return nil
	default:
		err = fmt.Errorf("json: cannot unmarshal %v into Go value of type null.Int64", reflect.TypeOf(v).Name())
	}
	i.Valid = err == nil
	return err
}

func (i *Int64) UnmarshalText(text []byte) error {
	str := string(text)
	if str == "" || str == "null" {
		i.Valid = false
		return nil
	}
	var err error
	i.Int64, err = parse64(string(text))
	i.Valid = err == nil
	return err
}

func parse64(str string) (int64, error) {
	v, err := strconv.ParseInt(str, 10, 64)
	return v, err
}

func (i Int64) MarshalJSON() ([]byte, error) {
	if !i.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(i.Int64), 10)), nil
}

func (i Int64) MarshalText() ([]byte, error) {
	if !i.Valid {
		return []byte{}, nil
	}
	return []byte(strconv.FormatInt(int64(i.Int64), 10)), nil
}

func (i *Int64) SetValid(n int64) {
	i.Int64 = n
	i.Valid = true
}

func (i Int64) Value() (driver.Value, error) {
	if !i.Valid {
		return nil, nil
	}

	return int64(i.Int64), nil
}

func (i *Int64) Scan(value interface{}) error {
	if value == nil {
		*i = Int64{}
		return nil
	}

	switch typed := value.(type) {
	case int:
		safe := int64(typed)
		*i = Int64From(safe)
	case int32:
		safe := int64(typed)
		*i = Int64From(safe)
	case int64:
		safe := int64(typed)
		*i = Int64From(safe)
	case uint:
		if typed > uint(math.MaxInt64) {
			return fmt.Errorf("unable to convert %v of %T to Int64; overflow", value, value)
		}
		safe := int64(typed)
		*i = Int64From(safe)
	case uint64:
		if typed > uint64(math.MaxInt64) {
			return fmt.Errorf("unable to convert %v of %T to Int64; overflow", value, value)
		}
		safe := int64(typed)
		*i = Int64From(safe)
	default:
		return fmt.Errorf("unable to convert %v of %T to Int64", value, value)
	}
	return nil
}
