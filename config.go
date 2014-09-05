package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config struct {
	SmzdmCron string `toml:"smzdm_cron"`
	KjlCron   string `toml:"kjl_cron"`
}

var Conf Config

//读取配置文件
func ReadConfig() error {
	if _, err := toml.DecodeFile("config.ini", &Conf); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
