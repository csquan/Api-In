package config

import (
	"fmt"
	"github.com/jialanli/windward"
	"os"
)

type Config struct {
	Db struct {
		Name     string `yaml:"name"`
		Password string `yaml:"password"`
		Ip       string `yaml:"ip"`
		Port     string `yaml:"port"`
		Database string `yaml:"database"`
	}
	Endpoint struct {
		Ip   string `yaml:"ip"`
		Port string `yaml:"port"`
	}
	TxState struct {
		EndPoint string `yaml:"endpoint"`
	}
	Pri struct {
		Value string `yaml:"value"`
	}
	Server struct {
		Port string `yaml:"port"`
	}
}

func Readconfig(filename string) (*Config, error) {
	path, err := os.Getwd()
	if err != nil {
		fmt.Errorf("++++++err ++++++++++: %v", err)
		return nil, fmt.Errorf("err : %v", err)
	}
	//加载配置文件
	file := path + "/" + filename
	w := windward.GetWindward()
	w.InitConf([]string{file}) //初始化自定义的配置文件

	//获取数据库连接名密码等数据
	var config Config //定义结构体【注意：这里需要有两层结构，因为w.ReadConfig读取的是data以及data中的数据】

	err = w.ReadConfig(file, &config)
	if err != nil {
		fmt.Sprintln("初始化配置文件失败")
		return nil, err
	}
	return &config, nil
}
