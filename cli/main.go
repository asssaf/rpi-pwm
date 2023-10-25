package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/asssaf/rpi-pwm-go/cli/cmd"
)

func main() {
	err := Execute()
	if err != nil {
		fmt.Fprintln(flag.CommandLine.Output(), err.Error())
		flag.Usage()
		os.Exit(1)
	}

	os.Exit(0)
}

func Execute() error {
	rootCommand := cmd.NewRootCommand("rpi-pwm")
	flag.Usage = rootCommand.FlagSet.Usage
	flag.Parse()

	if err := rootCommand.Init(flag.Args()); err != nil {
		return err
	}
	return rootCommand.Execute()
}
