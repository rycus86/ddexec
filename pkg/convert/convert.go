package convert

import (
	"fmt"
	"github.com/pkg/errors"
)

func ToStringSlice(v interface{}) []string {
	if v == nil {
		return []string{}
	} else if str, ok := v.(string); ok {
		return []string{str}
	} else if arr, ok := v.([]string); ok {
		return arr
	} else if varr, ok := v.([]interface{}); ok {
		var arr []string
		for _, item := range varr {
			arr = append(arr, item.(string))
		}
		return arr
	} else if m, ok := v.(map[string]string); ok {
		var arr []string
		for key, value := range m {
			arr = append(arr, fmt.Sprintf("%s=%s", key, value))
		}
		return arr
	} else if m, ok := v.(map[interface{}]interface{}); ok {
		var arr []string
		for key, value := range m {
			arr = append(arr, fmt.Sprintf("%s=%s", key, value))
		}
		return arr
	} else {
		panic(errors.New(fmt.Sprintf("unexpected type for string slice: %T", v)))
	}
}
