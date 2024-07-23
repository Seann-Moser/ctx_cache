package ctx_cache

import (
	"github.com/goccy/go-json"
	"reflect"
	"strconv"
)

func ConvertToBytes(data interface{}) ([]byte, error) {
	switch v := data.(type) {
	case string:
		return []byte(v), nil
	case int:
		return []byte(strconv.Itoa(v)), nil
	case int8:
		return []byte(strconv.Itoa(int(v))), nil
	case int16:
		return []byte(strconv.Itoa(int(v))), nil
	case int32:
		return []byte(strconv.Itoa(int(v))), nil
	case int64:
		return []byte(strconv.FormatInt(v, 10)), nil
	case uint:
		return []byte(strconv.FormatUint(uint64(v), 10)), nil
	case uint8:
		return []byte(strconv.Itoa(int(v))), nil
	case uint16:
		return []byte(strconv.FormatUint(uint64(v), 10)), nil
	case uint32:
		return []byte(strconv.FormatUint(uint64(v), 10)), nil
	case uint64:
		return []byte(strconv.FormatUint(v, 10)), nil
	case float32:
		return []byte(strconv.FormatFloat(float64(v), 'f', -1, 32)), nil
	case float64:
		return []byte(strconv.FormatFloat(v, 'f', -1, 64)), nil
	case bool:
		return []byte(strconv.FormatBool(v)), nil
	default:
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}

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
	default:
		// Attempt to unmarshal into the generic type
		err = json.Unmarshal(data, &result)
		if err != nil {
			return result, err
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
