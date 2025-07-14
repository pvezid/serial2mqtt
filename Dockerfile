FROM golang:1.24

WORKDIR /app
COPY src .
RUN go mod download
RUN go build -v -o serial2mqtt serial2mqtt.go

CMD ["/app/serial2mqtt", "-h", "tcp://mqtt:1883", "-p", "ser1/tx", "-s", "ser1/rx", "-d", "/dev/ttyACM0"]
