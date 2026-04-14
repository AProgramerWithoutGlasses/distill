package settings

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type AppConfig struct {
	Mode string `mapstructure:"mode"`
	Port int    `mapstructure:"port"`

	*LogConfig   `mapstructure:"log"`
	*MySQLConfig `mapstructure:"mysql"`
	*RedisConfig `mapstructure:"redis"`
}

type MySQLConfig struct {
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DB       string `mapstructure:"dbname"`
	Port     int    `mapstructure:"port"`
	//	MaxOpenConns int    `mapstructure:"max_open_conns"`
	//	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

// Init 主要作用是获取 local.yaml 文件的信息到返回值 app *AppConfig 上
func Init(cfgName string) (app *AppConfig, err error) {
	app = new(AppConfig)
	viper.SetConfigFile("configs/" + cfgName + ".yaml") // 指定配置文件为config.yaml
	err = viper.ReadInConfig()                          // 读取配置信息
	if err != nil {
		// 读取配置信息失败
		fmt.Printf("viper.ReadInConfig failed, err:%v\n", err)
		return
	}

	// 把读取到的配置信息反序列化到到app结构体对象中
	if err := viper.Unmarshal(app); err != nil {
		fmt.Printf("viper.Unmarshal failed, err:%v\n", err)
	}

	// 开启监听config文件
	viper.WatchConfig()

	// 若config文件发生变化，则自动触发该回调函数
	viper.OnConfigChange(func(in fsnotify.Event) { // in fsnotify.Event 参数为回调函数提供了关于文件变化的详细信息
		fmt.Println("配置文件修改了...")
		if err := viper.Unmarshal(app); err != nil { // 重新加载配置
			fmt.Printf("viper.Unmarshal failed, err:%v\n", err)
		}
	})
	return
}
