package config

import "github.com/BurntSushi/toml"

type Etcd struct {
	Ip   string `toml:"ip"`
	Port string `toml:"port"`
}

type Local struct {
	Ip   string `toml:"ip"`
	Port string `toml:"port"`
}

type Coordinator struct {
	Ip   string `toml:"ip"`
	Port string `toml:"port"`
}

type Resouce struct {
	Type string `toml:"type"`
	Kind string `toml:"kind"`
	Size int64  `toml:"size"`
	Unit string `toml:"unit"`
}

type Disk struct {
	Name    string    `toml:"name"`
	Level   string    `toml:"level"`
	Resouce []Resouce `toml:"resource"`
}

type Config struct {
	Disk        []Disk      `toml:"disk"`
	Etcd        []Etcd      `toml:"disk"`
	Local       Local       `toml:"local"`
	Coordinator Coordinator `toml:"coordinator"`
}

var c *Config = new(Config)
var Path string = "../../config.toml"

func Init(path string) {
	Path = path
	err := c.Decode()
	if err != nil {
		panic(err)
	}
}

func (c *Config) Decode() error {
	capath := Path
	if _, err := toml.DecodeFile(capath, c); err != nil {
		return err
	}
	return nil
}

func GetDisks() []Disk {
	return c.Disk
}

func GetEtcds() []Etcd {
	return c.Etcd
}

func GetLocal() Local {
	return c.Local
}

func GetCoordinator() Coordinator {
	return c.Coordinator
}
