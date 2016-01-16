//服务端设置项的结构体，读取config.ini映射到Config结构体中。
package main

import "github.com/go-ini/ini"

type Config struct {
	DB         DBConfig     `ini:"database"`
	Server     ServerConfig `ini:"server"`
	AuthConfig AuthConfig   `ini:"auth"`
}

type ServerConfig struct {
	Port       string `ini:"port"`
	StaticHome string `ini:"statichome"`
}

type AuthConfig struct {
	ConstSalt string `ini:"constsalt"`
}

type DBConfig struct {
	Uri  string `ini:"uri"`
	Type string `ini:"type"`
	Pwd  string `ini:"passwd"`
	Name string `ini:"dbname"`
	User string `ini:"user"`
}

func InitConfig() Config {
	cfg, err := ini.Load("config.ini")
	cfg.BlockMode = false
	if err != nil {
		panic(err)
	}
	config := Config{}
	err = cfg.MapTo(&config)
	if err != nil {
		panic(err)
	}

	return config
}
