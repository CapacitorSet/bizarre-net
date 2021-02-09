package main

import (
	"github.com/BurntSushi/toml"
	"io/ioutil"
)

type Config struct {
	TunPrefix string
	TunIP string
	ListenIP string
}

func ReadConfig(file string) (Config, error) {
	var config Config
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return Config{}, err
	}
	_, err = toml.Decode(string(content), &config)
	if err != nil {
		return Config{}, err
	}
	if config.TunPrefix == "" {
		config.TunPrefix = "bizarre"
	}
	return config, nil
}