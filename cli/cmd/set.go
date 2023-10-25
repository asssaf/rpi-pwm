package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"
)

type SetCommand struct {
	fs  *flag.FlagSet
	num int
}

func NewSetCommand(usagePrefix string) *SetCommand {
	c := &SetCommand{
		fs: flag.NewFlagSet("set", flag.ExitOnError),
	}

	c.fs.IntVar(&c.num, "num", 0, "PWM number (1-2)")

	c.fs.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s %s\n", usagePrefix, c.fs.Name())
		c.fs.PrintDefaults()
	}

	return c
}

func (c *SetCommand) Name() string {
	return c.fs.Name()
}

func (c *SetCommand) Init(args []string) error {
	if err := c.fs.Parse(args); err != nil {
		return err
	}

	flag.Usage = c.fs.Usage

	if c.num < 1 || c.num > 2 {
		return fmt.Errorf("PWM num must be in the range 1-2: %d", c.num)
	}

	return nil
}

func (c *SetCommand) Execute() error {
	if _, err := host.Init(); err != nil {
		log.Fatal(err)
	}

	pinName := fmt.Sprintf("PWM%d_OUT", c.num-1)
	p := gpioreg.ByName(pinName)
	if p == nil {
		log.Fatalf("Failed to find %s", pinName)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		str := scanner.Text()
		value, err := strconv.ParseFloat(str, 64)
		if err != nil || value < 0.0 || value > 1.0 {
			log.Printf("pwm duty value must be in the range 0.0-1.0: %s", str)
			continue
		}

		if err := c.writeSingleValue(p, value); err != nil {
			log.Fatalf("write pwm: %w", err)
		}
	}

	time.Sleep(1 * time.Second)

	if err := p.Halt(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func (c *SetCommand) writeSingleValue(p gpio.PinIO, value float64) error {
	// the range is 1ms to 2ms out of 20ms (in 50Hz), so between 5% and 10%
	duty := gpio.Duty((value/20 + 0.05) * float64(gpio.DutyMax))
	if err := p.PWM(duty, 50*physic.Hertz); err != nil {
		return err
	}
	return nil
}
