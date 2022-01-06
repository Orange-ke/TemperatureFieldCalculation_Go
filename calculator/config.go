package calculator

import (
	"fmt"
	"gopkg.in/ini.v1"
)

var calCfg Config

type Config struct {
	XStep   int
	YStep   int
	ZStep   int
	Length  int
	Width   int
	ZLength int

	ArrayLength int

	EdgeWidth int
}

func init() {
	file, err := ini.Load("../conf/config.ini")
	if err != nil {
		fmt.Println("配置文件读取错误，请检查文件路径: ", err)
	}

	loadCfg(file)
}

func loadCfg(file *ini.File) {
	calCfg = Config{
		XStep: file.Section("calculator").Key("XStep").MustInt(5),
		YStep: file.Section("calculator").Key("YStep").MustInt(5),
		ZStep: file.Section("calculator").Key("ZStep").MustInt(10),
		Length: file.Section("calculator").Key("Length").MustInt(1350),
		Width: file.Section("calculator").Key("Width").MustInt(210),
		ZLength: file.Section("calculator").Key("ZLength").MustInt(40000),
		ArrayLength: file.Section("calculator").Key("ArrayLength").MustInt(320),
		EdgeWidth: file.Section("calculator").Key("EdgeWidth").MustInt(40),
	}
}
