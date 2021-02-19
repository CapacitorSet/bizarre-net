package bizarre_net

import (
	"errors"
	"github.com/BurntSushi/toml"
)

type Config struct {
	TUN         TUNConfig `toml:"tun"`
	DropChatter bool
	Transport   string
	UDP         toml.Primitive
	Cat         toml.Primitive
	Socket      toml.Primitive
}

func ReadConfig(file string) (Config, toml.MetaData, error) {
	// Defaults
	config := Config{
		TUN: TUNConfig{
			Prefix: "bizarre",
		},
		DropChatter: true,
	}
	md, err := toml.DecodeFile(file, &config)
	if err != nil {
		return Config{}, toml.MetaData{}, err
	}
	if config.Transport == "" {
		return Config{}, toml.MetaData{}, errors.New("no transport selected")
	}
	return config, md, nil
}
