/*
  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <http://www.gnu.org/licenses/>.

  Copyright © 2024 Georges Ménie.
*/

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"menie.org/messager/handlers"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	brokerURL    string
	logFile      string
	logMaxSize   int
	logArchDir   string
	serialdev    string
	serialbaud   int
	tcpserver    string
	udpbroadcast string
	subtopic     string
	pubtopic     string
	debugmode    bool
)

func init() {
	flag.StringVar(&brokerURL, "h", "", "MQTT broker to use")
	flag.StringVar(&logFile, "l", "", "log data to file")
	flag.IntVar(&logMaxSize, "ls", 1000000, "log file max size")
	flag.StringVar(&logArchDir, "la", "/var/spool/messager", "log file archive directory")
	flag.StringVar(&serialdev, "d", "", "serial device to use (mandatory)")
	flag.IntVar(&serialbaud, "b", 115200, "serial baudrate")
	flag.StringVar(&tcpserver, "t", "localhost:5000", "TCP port")
	flag.StringVar(&udpbroadcast, "u", "", "UDP broadcast")
	flag.StringVar(&subtopic, "s", "", "topic to be subscribed")
	flag.StringVar(&pubtopic, "p", "", "topic to be published")
	flag.BoolVar(&debugmode, "debug", false, "set loglevel to DEBUG")
}

func main() {

	setFlags()
	setLogger()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	handlers.SerialPortList()

	ich, och := handlers.TeeHandler(logFile, int64(logMaxSize), logArchDir)

	go func() {

		// SerialHandler tournent en boucle en permanence
		_, sc1 := handlers.SerialHandler("/dev/ttyGPS", 4800)
		_, sc2 := handlers.SerialHandler("/dev/ttyGX2200E", 38400)

		state := 1
		monTimeout := 3 * time.Second
		tmr := time.AfterFunc(monTimeout, func() { state = 0 })

	Loop:
		for {
			select {
			case s := <-sc1:
				m := handlers.MaskableMessage{}
				m.Device = "/dev/ttyGPS"
				m.Msg = s
				m.Masked = (state == 1)
				ich <- m
			case s := <-sc2:
				tmr.Reset(monTimeout)
				state = 1
				m := handlers.MaskableMessage{}
				m.Device = "/dev/ttyGX2200E"
				m.Msg = s
				ich <- m
			case <-sig:
				slog.Info("Stop signal received")
				break Loop
			}
		}
		close(ich)
	}()

	_, c1 := handlers.MQTTHandler(brokerURL, subtopic, pubtopic)
	c2 := handlers.TCPHandler(tcpserver)
	c3 := handlers.UDPHandler(udpbroadcast)

	for msg := range och {
		if c1 != nil {
			c1 <- msg
		}
		if c2 != nil {
			c2 <- msg
		}
		if c3 != nil {
			c3 <- msg
		}
	}
}

func setFlags() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
}

func setLogger() {
	logLevel := &slog.LevelVar{} // INFO par défaut
	if debugmode {
		logLevel.Set(slog.LevelDebug)
	}
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, opts))
	slog.SetDefault(logger)
}
