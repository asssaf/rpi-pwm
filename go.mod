module github.com/asssaf/rpi-pwm-go

go 1.16

require (
	periph.io/x/conn/v3 v3.7.0
	periph.io/x/host/v3 v3.7.1
)

replace periph.io/x/host/v3 => github.com/asssaf/periphio-host v0.0.0-20231029031548-f22d1c777e36

replace periph.io/x/host/v3/sysfs => github.com/asssaf/periphio-host/sysfs v0.0.0-20231029031548-f22d1c777e36
