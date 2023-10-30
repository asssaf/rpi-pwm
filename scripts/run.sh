: ${IMAGE_NAME:=asssaf/rpi-pwm:latest}
docker run --rm -it --privileged --device /dev/gpiomem "$IMAGE_NAME" $*
