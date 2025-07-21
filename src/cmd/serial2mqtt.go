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
)

var (
	brokerURL  string
	serialdev  string
	serialbaud int
	subtopic   string
	pubtopic   string
	debugmode  bool
)

func init() {
	flag.StringVar(&brokerURL, "h", "tcp://mqtt:1883", "MQTT broker to use")
	flag.StringVar(&serialdev, "d", "", "serial device to use (mandatory)")
	flag.IntVar(&serialbaud, "b", 115200, "serial baudrate")
	flag.StringVar(&subtopic, "s", "", "topic to be subscribed")
	flag.StringVar(&pubtopic, "p", "", "topic to be published")
	flag.BoolVar(&debugmode, "debug", false, "set loglevel to DEBUG")
}

func main() {

	setFlags()
	setLogger()

	if serialdev == "" {
		slog.Error("A serial device is mandatory")
		os.Exit(1)
	}

	sic, soc := handlers.SerialHandler(serialdev, serialbaud, "", true)
	nic, noc := handlers.MQTTHandler(brokerURL, subtopic, pubtopic)

	go func() {
		for s := range soc {
			nic <- s
		}
	}()
	for s := range noc {
		sic <- s
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
