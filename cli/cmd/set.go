package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"
)

type SetCommand struct {
	fs  *flag.FlagSet
	num int

	moveIntervalDuration time.Duration
	maxMoveDuration      time.Duration
	moveStepSize         float64
	minInputValue        float64
	maxInputValue        float64
	minPulseDuration     time.Duration
	maxPulseDuration     time.Duration
	dryRun               bool

	target *float64

	mu sync.Mutex
}

func NewSetCommand(usagePrefix string) *SetCommand {
	c := &SetCommand{
		fs: flag.NewFlagSet("set", flag.ExitOnError),
	}

	c.fs.IntVar(&c.num, "num", 0, "PWM number (1-2)")
	c.fs.DurationVar(&c.moveIntervalDuration, "move-interval-duration", 5*time.Millisecond, "Move interval duration")
	c.fs.DurationVar(&c.maxMoveDuration, "max-move-duration", 500*time.Millisecond, "Maximum movement duration")
	c.fs.Float64Var(&c.minInputValue, "min-input-value", 0.0, "Input value for minimum position")
	c.fs.Float64Var(&c.maxInputValue, "max-input-value", 1.0, "Input value for maximum position")
	c.fs.DurationVar(&c.minPulseDuration, "min-pulse-duration", 1*time.Millisecond, "Pulse duration for min input value (setting to less than 1ms for servos is dangerous!)")
	c.fs.DurationVar(&c.maxPulseDuration, "max-pulse-duration", 2*time.Millisecond, "Pulse duration for max input value (setting to more than 2ms for servos is dangerous!)")
	c.fs.BoolVar(&c.dryRun, "dry-run", false, "Don't actually send the value to the PWM device")

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

	if c.minInputValue >= c.maxInputValue {
		return fmt.Errorf("min-input-value should be less than max-input-value: %f %f", c.minInputValue, c.maxInputValue)
	}

	if c.minPulseDuration >= c.maxPulseDuration {
		return fmt.Errorf("min-pulse-duration should be less than max-pulse-duration: %s %s", c.minPulseDuration, c.maxPulseDuration)
	}

	c.moveStepSize = float64(c.moveIntervalDuration) / float64(c.maxMoveDuration)

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

	defer p.Halt()

	// make sure the dma channel is released when the program is terminated, otherwise could run out of dma resources
	var halt = make(chan os.Signal)
	signal.Notify(halt, syscall.SIGTERM)
	signal.Notify(halt, syscall.SIGINT)

	go func() {
		select {
		case <-halt:
			if err := p.Halt(); err != nil {
				log.Println(err)
			}
			os.Exit(1)
		}
	}()

	// wait group for all launched goroutines
	wg := sync.WaitGroup{}

	// channel for notifying the loop that there is a new target in case it is idle
	// using a buffer of 1 so we can do a non blocking write and be sure that the
	// control loop will be triggered
	targetCh := make(chan struct{}, 1)

	// run the control loop in a goroutine
	go c.controlLoop(p, targetCh, &wg)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		str := scanner.Text()
		value, err := strconv.ParseFloat(str, 64)
		if err != nil || value < c.minInputValue || value > c.maxInputValue {
			log.Printf("pwm duty value must be in the range %f-%f: %s", c.minInputValue, c.maxInputValue, str)
			continue
		}

		valueNormalized := (value - c.minInputValue) / (c.maxInputValue - c.minInputValue)
		c.setTarget(valueNormalized)

		// non blocking write to the target channel to trigger the control loop if it's idle
		// but not wait if it's already running
		select {
		case targetCh <- struct{}{}:
			// triggered
		default:
			// busy
		}
	}

	// notify the control routine when there are no more new targets
	close(targetCh)

	// wait for the goroutines to finish
	wg.Wait()

	time.Sleep(1 * time.Second)

	if err := p.Halt(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func (c *SetCommand) writeSingleValue(p gpio.PinIO, value float64) error {
	// the range is 1ms to 2ms out of 20ms (in 50Hz), so between 5% and 10%
	period := 20 * time.Millisecond // (at 50Hz)
	minPulseDuty := float64(c.minPulseDuration) / float64(period)
	maxPulseDuty := float64(c.maxPulseDuration) / float64(period)
	duty := gpio.Duty((minPulseDuty + value*(maxPulseDuty-minPulseDuty)) * float64(gpio.DutyMax))
	if c.dryRun {
		fmt.Printf("dry-run: would write %s to PWM device\n", duty)
		return nil
	}

	if err := p.PWM(duty, 50*physic.Hertz); err != nil {
		return err
	}
	return nil
}

func (c *SetCommand) moverLoop(p gpio.PinIO, valueCh <-chan float64, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	// move to position values coming in the channel, limiting the rate by sleeping after every move
	// exits when the channel is closed
	for value := range valueCh {
		if err := c.writeSingleValue(p, value); err != nil {
			return err
		}

		// wait a minimum amount of time before moving again
		time.Sleep(c.moveIntervalDuration)
	}
	return nil
}

func (c *SetCommand) controlLoop(p gpio.PinIO, targetCh <-chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// last known position
	var current *float64

	// create the moverLoop goroutine
	moveCh := make(chan float64)
	defer close(moveCh)
	go c.moverLoop(p, moveCh, wg)

	// will be set to true when the target channel is closed
	done := false

	for {
		target, ok := c.getTarget()

		if ok && (current == nil || *current != target) {
			// move
			if current == nil || math.Abs(*current-target) < c.moveStepSize {
				moveCh <- target
				current = &target
			} else {
				var nextValue float64
				if *current < target {
					nextValue = *current + c.moveStepSize
				} else if *current > target {
					nextValue = *current - c.moveStepSize
				}
				moveCh <- nextValue
				current = &nextValue
			}

		} else {
			if done {
				break
			}

			// wait for new target
			_, ok := <-targetCh
			if !ok {
				// set done to true, but don't exit yet since we might still need to move to the last target
				done = true
			}
		}
	}
}

func (c *SetCommand) setTarget(newTarget float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.target = &newTarget
}

func (c *SetCommand) getTarget() (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.target == nil {
		return 0, false
	}

	return *c.target, true
}
