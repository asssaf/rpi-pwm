# rpi-pwm
periph.io based cli for driving gpio pwm

## Usage
### /boot/firmware/config.txt
Enable the desired pwm overlay in, for example:
```
dtoverlay=pwm-2chan,pin=12,func=4,pin2=13,func2=4
```

Make sure that audio is enabled:
```
dtparam=audio=on
```

### Run: 
```
$ echo "0.5" | rpi-pwm set -num 1
```
Values between 0 and 1 can be provided on standard input.

### Running in docker
The bcm283x gpio driver uses /dev/gpiomem for gpio access, but also needs DMA for PWM. This means docker containers need to be run in privileged mode (with --privileged):

```
$ docker run --privileged --device /dev/gpiomem ...
```

This CLI uses [a forked version of periph.io/host](https://github.com/asssaf/periphio-host/pull/1) that supports PWM through sysfs when DMA 
isn't available (when running without --privileged). 

It also uses files in /sys/firmware (to map gpio pin numbers to pwm numbers), which docker doesn't export by 
default, so additional security-opts are needed to export them. Alternatively, these files can be manually copied into the container.
```
$ docker run --user nobody:gpio --device /dev/gpiomem -v /sys:/sys \
  --security-opt systempaths=unconfined --security-opt apparmor=unconfined ...
```
