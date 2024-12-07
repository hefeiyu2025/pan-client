package internal

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"reflect"
	"strconv"
)

type ServerConfig struct {
	Debug           bool   `mapstructure:"debug" json:"debug" yaml:"debug" default:"false"`
	CacheFile       string `mapstructure:"cache_file" json:"cache_file"  yaml:"cache_file"`
	DownloadTmpPath string `mapstructure:"download_tmp_path" json:"download_tmp_path"  yaml:"download_tmp_path"`
}

type LogConfig struct {
	Enable     bool   `mapstructure:"enable" json:"enable" yaml:"enable" default:"true"`
	FileName   string `mapstructure:"file_name" json:"file_name" yaml:"file_name" default:"app.log"`
	MaxSize    int    `mapstructure:"max_size" json:"max_size"  yaml:"max_size" default:"50"`
	MaxBackups int    `mapstructure:"max_backups" json:"max_backups" yaml:"max_backups" default:"30"`
	MaxAge     int    `mapstructure:"max_age" json:"max_age"  yaml:"max_age" default:"28"`
	Compress   bool   `mapstructure:"compress" json:"compress" yaml:"compress" default:"false"`
}

type RootConfig struct {
	Server *ServerConfig `mapstructure:"server" json:"server" yaml:"server"`
	Log    *LogConfig    `mapstructure:"log" json:"log" yaml:"log"`
}

var Config RootConfig

func InitConfig() {
	configName := "pan-client"
	// 添加运行目录

	viper.AddConfigPath(GetProcessPath())

	// 添加当前目录
	viper.AddConfigPath(GetWordPath())
	viper.SetConfigName(configName)
	SetDefaultByTag(&Config)
	if err := viper.ReadInConfig(); err != nil { // 读取配置文件
		// 使用类型断言检查是否为 *os.PathError 类型
		var pathErr viper.ConfigFileNotFoundError
		if errors.As(err, &pathErr) {
			v := reflect.ValueOf(Config)
			for i := 0; i < v.NumField(); i++ {
				field := v.Field(i)
				// 获取字段名
				name := v.Type().Field(i).Name
				// 获取字段值
				value := field.Interface()
				viper.SetDefault(name, value)
			}
			err = viper.WriteConfigAs(GetWordPath() + "/" + configName + ".yaml")
			if err != nil {
				panic(err)
			} else {
				// 重新读取已经写入的文件
				_ = viper.ReadInConfig()
			}
		} else {
			panic(err)
		}
	}

	if err := viper.Unmarshal(&Config); err != nil { // 解码配置文件到结构体
		panic(err)
	}
}

// SetDefaultByTag 根据结构体字段的tag设置默认值，包括嵌套对象和指针
func SetDefaultByTag(obj interface{}) {
	// 获取对象的反射值
	v := reflect.ValueOf(obj)
	// 确保对象是可设置的（非指针或指针指向的值）
	if v.Kind() != reflect.Ptr || !v.Elem().CanSet() {
		panic("SetDefaultByTag requires a pointer to a struct")
	}
	v = v.Elem()

	// 递归遍历结构体的所有字段
	setDefaults(v)
}

// setDefaults 递归设置默认值
func setDefaults(v reflect.Value) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue // 如果字段不可设置，则跳过
		}
		if field.Kind() == reflect.Ptr {
			// 如果字段是指针类型，需要特别处理
			// 尝试创建一个新的实例
			ptrValue := reflect.New(field.Type().Elem())
			if ptrValue.Type() == field.Type() {
				// 递归设置指针指向的值
				setDefaults(ptrValue.Elem())
				// 将指针指向新的实例
				field.Set(ptrValue)
			}
			continue
		} else if field.Kind() == reflect.Struct {
			// 如果是结构体，则递归调用setDefaults
			setDefaults(field)
			continue
		}

		// 获取字段的tag
		tag := v.Type().Field(i).Tag
		defaultValue := tag.Get("default") // 从tag中获取默认值

		// 如果有默认值，则设置
		if defaultValue != "" {
			switch field.Kind() {
			case reflect.String:
				field.SetString(defaultValue)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				val, err := strconv.ParseInt(defaultValue, 10, 64)
				if err == nil {
					field.SetInt(val)
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				val, err := strconv.ParseUint(defaultValue, 10, 64)
				if err == nil {
					field.SetUint(val)
				}
			case reflect.Float32, reflect.Float64:
				val, err := strconv.ParseFloat(defaultValue, 64)
				if err == nil {
					field.SetFloat(val)
				}
			case reflect.Bool:
				val, err := strconv.ParseBool(defaultValue)
				if err == nil {
					field.SetBool(val)
				}
			default:
				// 其他类型暂不支持
				fmt.Printf("Unsupported type for field %s\n", v.Type().Field(i).Name)
			}
		}
	}
}
