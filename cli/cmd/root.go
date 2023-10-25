package cmd

import (
	"flag"

	"github.com/asssaf/rpi-pwm-go/cli/util"
)

type Command = util.Command

type RootCommand struct {
	*util.CompositeCommand
}

func NewRootCommand(usagePrefix string) *RootCommand {
	c := &RootCommand{
		CompositeCommand: util.NewCompositeCommand(
			flag.NewFlagSet(usagePrefix, flag.ExitOnError),
			[]Command{
				NewSetCommand(""),
			},
			"",
		),
	}

	return c
}
