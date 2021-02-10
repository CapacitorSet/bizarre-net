package bizarre_net

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	TUN TUNConfig `toml:"tun"`
	UDP toml.Primitive
	Cat toml.Primitive
}

func ReadConfig(file string) (Config, toml.MetaData, error) {
	var config Config
	md, err := toml.DecodeFile(file, &config)
	if err != nil {
		return Config{}, toml.MetaData{}, err
	}
	if config.TUN.Prefix == "" {
		config.TUN.Prefix = "bizarre"
	}
	return config, md, nil
}
