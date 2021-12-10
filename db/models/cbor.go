package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/fxamacker/cbor/v2"
)

func ParseCBOR(b []byte) (JSON, error) {
	if len(b) == 0 {
		return JSON{}, nil
	}

	var m map[interface{}]interface{}

	if err := cbor.Unmarshal(autoAddMapDelimiters(b), &m); err != nil {
		return JSON{}, err
	}

	coerced, err := CoerceInterfaceMapToStringMap(m)
	if err != nil {
		return JSON{}, err
	}

	jsb, err := json.Marshal(coerced)
	if err != nil {
		return JSON{}, err
	}

	var js JSON
	return js, json.Unmarshal(jsb, &js)
}

func autoAddMapDelimiters(b []byte) []byte {
	if len(b) < 2 {
		return b
	}

	if (b[0] >> 5) != 5 {
		var buffer bytes.Buffer
		buffer.Write([]byte{0xbf})
		buffer.Write(b)
		buffer.Write([]byte{0xff})
		return buffer.Bytes()
	}

	return b
}

func CoerceInterfaceMapToStringMap(in interface{}) (interface{}, error) {
	switch typed := in.(type) {
	case map[string]interface{}:
		for k, v := range typed {
			coerced, err := CoerceInterfaceMapToStringMap(v)
			if err != nil {
				return nil, err
			}
			typed[k] = coerced
		}
		return typed, nil
	case map[interface{}]interface{}:
		m := map[string]interface{}{}
		for k, v := range typed {
			coercedKey, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("unable to coerce key %T %v to a string", k, k)
			}
			coerced, err := CoerceInterfaceMapToStringMap(v)
			if err != nil {
				return nil, err
			}
			m[coercedKey] = coerced
		}
		return m, nil
	case []interface{}:
		r := make([]interface{}, len(typed))
		for i, v := range typed {
			coerced, err := CoerceInterfaceMapToStringMap(v)
			if err != nil {
				return nil, err
			}
			r[i] = coerced
		}
		return r, nil
	case big.Int:
		value, _ := (in).(big.Int)
		return &value, nil
	default:
		return in, nil
	}
}
