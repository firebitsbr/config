package config

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const delim = "__"

type Builder interface {
	FromEnv() Builder
	From(file string) Builder
	To(target interface{})
}

type configBuilder struct {
	delim     string
	configMap map[string]string
}

func newConfigBuilder() Builder {
	return &configBuilder{
		configMap: make(map[string]string),
		delim:     delim,
	}
}

func (c *configBuilder) mergeConfig(in map[string]string) {
	for k, v := range in {
		c.configMap[k] = v
	}
}

func From(file string) Builder {
	return newConfigBuilder().From(file)
}

func (c *configBuilder) From(f string) Builder {
	file, err := os.Open(f)
	if err != nil {
		panic(fmt.Sprintf("oops!: %v", err))
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var ss []string
	for scanner.Scan() {
		ss = append(ss, scanner.Text())
	}
	c.mergeConfig(stringsToMap(ss))
	return c
}

func FromEnv() Builder {
	return newConfigBuilder().FromEnv()
}

func stringsToMap(ss []string) map[string]string {
	m := make(map[string]string)
	for _, s := range ss {
		if !strings.Contains(s, "=") {
			continue // ensures return is always of length 2
		}
		split := strings.SplitN(s, "=", 2)
		key, value := split[0], split[1]
		if key != "" && value != "" {
			m[key] = value
		}
	}
	return m
}

func (c *configBuilder) FromEnv() Builder {
	c.mergeConfig(stringsToMap(os.Environ()))
	return c
}

func (c *configBuilder) To(target interface{}) {
	c.populateStructRecursively(target, "")
}

// populateStructRecursively populates each field of the passed in struct.
// slices and values are set directly.
// nested structs recurse through this function.
// values are derived from the field name, prefixed with the field names of any parents.
func (c *configBuilder) populateStructRecursively(structPtr interface{}, prefix string) {
	structValue := reflect.ValueOf(structPtr).Elem()
	for i := 0; i < structValue.NumField(); i++ {
		fieldType := structValue.Type().Field(i)
		fieldPtr := structValue.Field(i).Addr().Interface()

		key := prefix + fieldType.Name
		value := c.configMap[key]

		switch fieldType.Type.Kind() {
		case reflect.Struct:
			c.populateStructRecursively(fieldPtr, key+c.delim)
		case reflect.Slice:
			convertAndSetSlice(fieldPtr, stringToSlice(value))
		default:
			convertAndSetValue(fieldPtr, value)
		}
	}
}

// stringToSlice converts a space delimited string to a slice of string.
// It strips surrounding whitespace of all entries.
// If the input string is empty or all whitespace, nil is returned.
func stringToSlice(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	split := strings.Split(s, " ")
	filtered := split[:0] // https://github.com/golang/go/wiki/SliceTricks#filtering-without-allocating
	for _, v := range split {
		if v != "" {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// convertAndSetSlice builds a slice of a dynamic type.
// It converts each entry in "values" to the elemType of the passed in slice.
// The slice remains nil if "values" is empty.
func convertAndSetSlice(slicePtr interface{}, values []string) {
	sliceVal := reflect.ValueOf(slicePtr).Elem()
	elemType := sliceVal.Type().Elem()

	for _, s := range values {
		valuePtr := reflect.New(elemType)
		convertAndSetValue(valuePtr.Interface(), s)
		sliceVal.Set(reflect.Append(sliceVal, valuePtr.Elem()))
	}
}

// convertAndSetValue receives a settable of an arbitrary kind, and sets its value to s".
// It calls the matching strconv function on s, based on the settable's kind.
// All basic types (bool, int, float, string) are handled by this function.
// Slice and struct are handled elsewhere.
// Unhandled kinds panic.
// Errors in string conversion are ignored, and the settable remains a zero value.
func convertAndSetValue(settable interface{}, s string) {
	settableValue := reflect.ValueOf(settable).Elem()
	switch settableValue.Kind() {
	case reflect.String:
		settableValue.SetString(s)
	case reflect.Int:
		val, _ := strconv.ParseInt(s, 10, 0)
		settableValue.SetInt(val)
	case reflect.Int8:
		val, _ := strconv.ParseInt(s, 10, 8)
		settableValue.SetInt(val)
	case reflect.Int16:
		val, _ := strconv.ParseInt(s, 10, 26)
		settableValue.SetInt(val)
	case reflect.Int32:
		val, _ := strconv.ParseInt(s, 10, 32)
		settableValue.SetInt(val)
	case reflect.Int64:
		val, _ := strconv.ParseInt(s, 10, 64)
		settableValue.SetInt(val)
	case reflect.Uint:
		val, _ := strconv.ParseUint(s, 10, 0)
		settableValue.SetUint(val)
	case reflect.Uint8:
		val, _ := strconv.ParseUint(s, 10, 8)
		settableValue.SetUint(val)
	case reflect.Uint16:
		val, _ := strconv.ParseUint(s, 10, 16)
		settableValue.SetUint(val)
	case reflect.Uint32:
		val, _ := strconv.ParseUint(s, 10, 32)
		settableValue.SetUint(val)
	case reflect.Uint64:
		val, _ := strconv.ParseUint(s, 10, 64)
		settableValue.SetUint(val)
	case reflect.Bool:
		val, _ := strconv.ParseBool(s)
		settableValue.SetBool(val)
	case reflect.Float32:
		val, _ := strconv.ParseFloat(s, 32)
		settableValue.SetFloat(val)
	case reflect.Float64:
		val, _ := strconv.ParseFloat(s, 64)
		settableValue.SetFloat(val)
	default:
		panic(fmt.Sprintf("cannot handle kind %v\n", settableValue.Type().Kind()))
	}
}
