package commands

import (
	"github.com/urfave/cli/v3"
)

var Commands []*cli.Command

func Append(cmd *cli.Command) {
	Commands = append(Commands, cmd)
}
