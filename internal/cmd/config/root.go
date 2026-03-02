package config_cmd

import (
	"fmt"
	"jpy-cli/pkg/config"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "管理 jpy-config.yaml 配置参数",
	}

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newListCmd())

	return cmd
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出所有配置参数",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Reload from file to show persisted config
			cfg := loadRawConfig()
			if cfg == nil {
				return fmt.Errorf("无法加载配置")
			}
			
			data, _ := yaml.Marshal(cfg)
			fmt.Println(string(data))
			return nil
		},
	}
}

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "获取指定配置参数",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			cfg := loadRawConfig()
			if cfg == nil {
				return fmt.Errorf("无法加载配置")
			}

			val, found := getField(cfg, key)
			if !found {
				return fmt.Errorf("配置项不存在: %s", key)
			}
			fmt.Printf("%s: %v\n", key, val)
			return nil
		},
	}
}

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "设置配置参数",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			valStr := args[1]

			cfg := loadRawConfig()
			if cfg == nil {
				return fmt.Errorf("无法加载配置")
			}

			if err := setField(cfg, key, valStr); err != nil {
				return err
			}

			return config.SaveSettings(cfg)
		},
	}
}

// Helper to interact with Settings struct dynamically
func getField(s *config.Settings, key string) (interface{}, bool) {
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

func setField(s *config.Settings, key string, valStr string) error {
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

func loadRawConfig() *config.Settings {
	// Re-implement load just for settings to avoid circular deps or complex logic
	// But ideally we should use pkg/config helpers.
	// pkg/config/store.go handles 'Config' struct (servers), but root.go handles 'Settings' (jpy-config.yaml).
	// This separation is a bit messy. 
	// The user wants to manage jpy-config.yaml params.
	// We need a helper in pkg/config to load/save Settings specifically.
	
	return config.LoadSettings()
}
