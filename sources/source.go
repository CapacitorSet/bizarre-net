package sources

import (
	"flag"
	"fmt"
	"log"
)

type Source interface {
	Start(chan []byte)
	Write([]byte) error
}

type SourceConfig struct {
	TUNConfig TUNConfig
	CmdExecConfig CmdExecConfig
}

// PartialConfigFromFlags binds a flagset to a SourceConfig struct, so that the config is filled upon parsing the flags.
func PartialConfigFromFlags(config *SourceConfig, flags *flag.FlagSet) {
	flags.StringVar(&config.TUNConfig.Name, "tun", "", "Name of the TUN interface (allows general-purpose navigation; requires root)")
	flags.StringVar(&config.TUNConfig.IP, "tun-ip", "", "TUN address in subnet form (eg. 192.168.100.1/24)")
	flags.BoolVar(&config.TUNConfig.DefaultRoute, "default-route", true, "Route all traffic to the TUN interface")

	flags.StringVar(&config.CmdExecConfig.Command, "cmd", "", "Command to run on the remote host")
}

// NewSource creates a Source from a SourceConfig.
func NewSource(config SourceConfig) (Source, error) {
	// todo: handle multiple sources
	if config.TUNConfig.Name != "" {
		tun, err := CreateTUN(config.TUNConfig)
		if err != nil {
			return nil, err
		}
		log.Printf("New interface: %s with IP %s", tun.Name, tun.IP.String())
		return &tun, nil
	} else if config.CmdExecConfig.Command != "" {
		cmd, err := CreateCmdExec(config.CmdExecConfig)
		if err != nil {
			return nil, err
		}
		log.Printf("Running command %s", cmd.Command)
		return &cmd, nil
	} else {
		return nil, fmt.Errorf("no source selected")
	}
}