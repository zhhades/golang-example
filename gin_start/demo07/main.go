package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"time"
)

type SeverConfig struct {
	Name        string      `mapstructure:"name"`
	MysqlConfig MysqlConfig `mapstructure:"mysql"`
}

type MysqlConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

func getEnv(env string) bool {
	viper.AutomaticEnv()
	return viper.GetBool(env)
}

func main() {
	v := viper.New()

	configFileName := "config_pro.yaml"
	if getEnv("DEBUG") {
		configFileName = "config_dev.yaml"
	}

	v.SetConfigFile("gin_start/demo07/" + configFileName)

	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}
	severConfig := SeverConfig{}
	if err := v.Unmarshal(&severConfig); err != nil {
		panic(err)
	}
	fmt.Println(severConfig)

	go func() {
		for {
			v.WatchConfig()
			v.OnConfigChange(func(in fsnotify.Event) {
				fmt.Println("config file changed:", in.Name)
				v.ReadInConfig()
				v.Unmarshal(severConfig)
				fmt.Println(severConfig)
			})
			time.Sleep(time.Second)
		}
	}()

	time.Sleep(1000 * time.Second)
}
