package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// GetField retrieves a field from Settings struct dynamically by its yaml tag key
func GetField(s *Settings, key string) (interface{}, bool) {
	v := reflect.ValueOf(s).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("yaml")
		// remove options like ",omitempty"
		tagName := strings.Split(tag, ",")[0]
		if tagName == key {
			return v.Field(i).Interface(), true
		}
	}
	return nil, false
}

// SetField sets a field in Settings struct dynamically by its yaml tag key
func SetField(s *Settings, key string, valStr string) error {
	v := reflect.ValueOf(s).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("yaml")
		tagName := strings.Split(tag, ",")[0]
		if tagName == key {
			f := v.Field(i)
			switch f.Kind() {
			case reflect.String:
				f.SetString(valStr)
			case reflect.Int:
				intVal, err := strconv.Atoi(valStr)
				if err != nil {
					return fmt.Errorf("无效的整数值: %s", valStr)
				}
				f.SetInt(int64(intVal))
			case reflect.Bool:
				boolVal, err := strconv.ParseBool(valStr)
				if err != nil {
					return fmt.Errorf("无效的布尔值: %s", valStr)
				}
				f.SetBool(boolVal)
			default:
				return fmt.Errorf("不支持的类型: %s", f.Kind())
			}
			return nil
		}
	}
	return fmt.Errorf("配置项不存在: %s", key)
}
