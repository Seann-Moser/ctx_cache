package ctx_cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

func ConvertToBytes(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case string:
		return []byte(v), nil
	case int:
		return strconv.AppendInt(nil, int64(v), 10), nil
	case int8, int16, int32, int64:
		return strconv.AppendInt(nil, toInt64(v), 10), nil
	case uint, uint8, uint16, uint32, uint64:
		return strconv.AppendUint(nil, toUint64(v), 10), nil
	case float32:
		return strconv.AppendFloat(nil, float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.AppendFloat(nil, v, 'f', -1, 64), nil
	case bool:
		return strconv.AppendBool(nil, v), nil
	case sql.NullString:
		if v.Valid {
			return []byte(v.String), nil
		}
		return nil, nil
	case sql.NullInt64:
		if v.Valid {
			return strconv.AppendInt(nil, v.Int64, 10), nil
		}
		return nil, nil
	case sql.NullFloat64:
		if v.Valid {
			return strconv.AppendFloat(nil, v.Float64, 'f', -1, 64), nil
		}
		return nil, nil
	case sql.NullBool:
		if v.Valid {
			return strconv.AppendBool(nil, v.Bool), nil
		}
		return nil, nil
	default:
		if data == nil {
			return nil, nil
		}
		b, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed converting data to bytes(%v): %w", data, err)
		}
		return b, nil
	}
}

func toInt64(data interface{}) int64 {
	switch v := data.(type) {
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

func toUint64(data interface{}) uint64 {
	switch v := data.(type) {
	case uint8:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v
	default:
		return 0
	}
}

// ConvertBytesToType attempts to convert a []byte to a generic type T
// ConvertBytesToType attempts to convert a []byte to a generic type T
func ConvertBytesToType[T any](data []byte) (T, error) {
	var result T
	var err error

	switch GetTypeReflect[T]() {
	case "int":
		var intValue int64
		intValue, err = strconv.ParseInt(string(data), 10, 64)
		result = any(int(intValue)).(T)
	case "int8":
		var intValue int64
		intValue, err = strconv.ParseInt(string(data), 10, 8)
		result = any(int8(intValue)).(T)
	case "int16":
		var intValue int64
		intValue, err = strconv.ParseInt(string(data), 10, 16)
		result = any(int16(intValue)).(T)
	case "int32":
		var intValue int64
		intValue, err = strconv.ParseInt(string(data), 10, 32)
		result = any(int32(intValue)).(T)
	case "int64":
		var intValue int64
		intValue, err = strconv.ParseInt(string(data), 10, 64)
		result = any(intValue).(T)
	case "uint":
		var uintValue uint64
		uintValue, err = strconv.ParseUint(string(data), 10, 64)
		result = any(uint(uintValue)).(T)
	case "uint8":
		var uintValue uint64
		uintValue, err = strconv.ParseUint(string(data), 10, 8)
		result = any(uint8(uintValue)).(T)
	case "uint16":
		var uintValue uint64
		uintValue, err = strconv.ParseUint(string(data), 10, 16)
		result = any(uint16(uintValue)).(T)
	case "uint32":
		var uintValue uint64
		uintValue, err = strconv.ParseUint(string(data), 10, 32)
		result = any(uint32(uintValue)).(T)
	case "uint64":
		var uintValue uint64
		uintValue, err = strconv.ParseUint(string(data), 10, 64)
		result = any(uintValue).(T)
	case "float32":
		var floatValue float64
		floatValue, err = strconv.ParseFloat(string(data), 32)
		result = any(float32(floatValue)).(T)
	case "float64":
		var floatValue float64
		floatValue, err = strconv.ParseFloat(string(data), 64)
		result = any(floatValue).(T)
	case "string":
		result = any(string(data)).(T)
	case "bool":
		var boolValue bool
		boolValue, err = strconv.ParseBool(string(data))
		result = any(boolValue).(T)
	case "sql.NullString":
		var nullString sql.NullString
		if len(data) == 0 {
			nullString.Valid = false
		} else {
			nullString.String = string(data)
			nullString.Valid = true
		}
		result = any(nullString).(T)
	case "sql.NullInt64":
		var nullInt64 sql.NullInt64
		if len(data) == 0 {
			nullInt64.Valid = false
		} else {
			var intValue int64
			intValue, err = strconv.ParseInt(string(data), 10, 64)
			nullInt64.Int64 = intValue
			nullInt64.Valid = err == nil
		}
		result = any(nullInt64).(T)
	case "sql.NullFloat64":
		var nullFloat64 sql.NullFloat64
		if len(data) == 0 {
			nullFloat64.Valid = false
		} else {
			var floatValue float64
			floatValue, err = strconv.ParseFloat(string(data), 64)
			nullFloat64.Float64 = floatValue
			nullFloat64.Valid = err == nil
		}
		result = any(nullFloat64).(T)
	case "sql.NullBool":
		var nullBool sql.NullBool
		if len(data) == 0 {
			nullBool.Valid = false
		} else {
			var boolValue bool
			boolValue, err = strconv.ParseBool(string(data))
			nullBool.Bool = boolValue
			nullBool.Valid = err == nil
		}
		result = any(nullBool).(T)
	default:
		if data == nil {
			return result, nil
		}
		// Attempt to unmarshal into the generic type
		err = json.Unmarshal(data, &result)
		if err != nil {
			return result, fmt.Errorf("failed converting bytes to type(%v): %w", GetTypeReflect[T](), err)
		}
	}
	return result, err
}

func CheckPrimaryType[T any](val T) bool {
	switch GetTypeReflect[T]() {
	case "int", "int8", "int16", "int32", "int64":
		return true
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	case "float32", "float64":
		return true
	case "string":
		return true
	case "bool":
		return true
	default:
		return false
	}
}

// GetTypeReflect returns the type of a generic value using reflect
func GetTypeReflect[T any]() string {
	return reflect.TypeOf(new(T)).String()
}
