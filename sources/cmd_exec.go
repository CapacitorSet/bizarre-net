package sources

import (
	"fmt"
	"io"
	"log"
)

const (
	CMD_EXEC_CMD_HEADER = byte(0xcd)
	CMD_EXEC_STDOUT_HEADER = byte(0xce)
)

type CmdExecConfig struct {
	Command string
}

type CmdExecSource struct {
	CmdExecConfig
}

func (S *CmdExecSource) Start(ch chan []byte) {
	log.Printf("Sending command %s", S.Command)
	msg := append([]byte{CMD_EXEC_CMD_HEADER}, []byte(S.Command)...)
	ch <- msg
}

func (S *CmdExecSource) Write(buf []byte) error {
	panic("CmdExecSource.Write is not implemented")
	return nil
}

var (
	_ Source = (*CmdExecSource)(nil) // Ensure that interface fields are implemented
)


// CreateCmdExec creates a CmdExec with the given config, if Name != "".
func CreateCmdExec(config CmdExecConfig) (CmdExecSource, error) {
	if config.Command == "" {
		return CmdExecSource{}, fmt.Errorf("empty command")
	}

	return CmdExecSource{config}, nil
}

type StdoutAdder struct {
	writer io.Writer
}

func (a StdoutAdder) Write(p []byte) (int, error) {
	n, err := a.writer.Write(append([]byte{CMD_EXEC_STDOUT_HEADER}, p...))
	return n-1, err
}

func WithStdoutHeader(w io.Writer) io.Writer {
	return StdoutAdder{w}
}