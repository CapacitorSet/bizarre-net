package bizarre_net

import (
	"errors"
	"github.com/BurntSushi/toml"
)

type Config struct {
	Transport        string
	DropChatter      bool
	SkipRoutingCheck bool
	TUN              TUNConfig `toml:"tun"`
	UDP              toml.Primitive
	Cat              toml.Primitive
	Socket           toml.Primitive
}

func ReadConfig(file string) (Config, toml.MetaData, error) {
	// Defaults
	config := Config{
		TUN: TUNConfig{
			Prefix:       "bizarre",
			SetDefaultGW: true,
		},
		DropChatter:      true,
		SkipRoutingCheck: false,
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
