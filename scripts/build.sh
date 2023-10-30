: ${IMAGE_NAME:=asssaf/rpi-pwm:latest}
BASE="$(dirname $0)/.."
docker build -t $IMAGE_NAME -f $BASE/docker/Dockerfile $BASE
