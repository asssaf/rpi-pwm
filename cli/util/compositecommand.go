package util

import (
	"errors"
	"flag"
	"fmt"
)

type CompositeCommand struct {
	FlagSet     *flag.FlagSet
	Commands    []Command
	UsagePrefix string
	Args        []string
}

func NewCompositeCommand(fs *flag.FlagSet, subcommands []Command, usagePrefix string) *CompositeCommand {
	c := &CompositeCommand{
		FlagSet:     fs,
		Commands:    subcommands,
		UsagePrefix: usagePrefix,
	}

	c.FlagSet.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s %s <command> ...\n", usagePrefix, c.FlagSet.Name())
		fmt.Fprintf(flag.CommandLine.Output(), "The commands are:\n")
		for _, c := range c.Commands {
			fmt.Fprintf(flag.CommandLine.Output(), "\t%s\n", c.Name())
		}
		flag.PrintDefaults()
	}

	return c
}

func (c *CompositeCommand) Name() string {
	return c.FlagSet.Name()
}

func (c *CompositeCommand) Init(args []string) error {
	if err := c.FlagSet.Parse(args); err != nil {
		return err
	}

	flag.Usage = c.FlagSet.Usage

	return nil
}

func (c *CompositeCommand) Execute() error {
	args := c.FlagSet.Args()
	flag := c.FlagSet

	command := flag.Arg(0)
	if command == "" {
		return errors.New("Missing command")
	}

	for _, c := range c.Commands {
		if command == c.Name() {
			if err := c.Init(args[1:]); err != nil {
				return err
			}
			return c.Execute()
		}
	}

	return errors.New(fmt.Sprintf("Unknown command: %s", command))
}
