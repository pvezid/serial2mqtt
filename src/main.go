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
	"sync"
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
	var twg sync.WaitGroup

	setFlags()
	setLogger()

	if serialdev == "" {
		slog.Error("A serial device is mandatory")
		os.Exit(1)
	}

	handlers.SerialPortList()

	ch_S2N := make(chan string, 10)
	ch_N2S := make(chan string, 3)

	// SerialHandler tourne en boucle en permanence
	if logFile == "" {
		go handlers.SerialHandler("/dev/ttyGPS", 4800, nil, ch_S2N, "$GP")
		go handlers.SerialHandler("/dev/ttyGX2200E", 38400, nil, ch_S2N, "!AI")
	} else {
		ch_log := make(chan string, 10)
		go handlers.SerialHandler("/dev/ttyGPS", 4800, nil, ch_log, "$GP")
		go handlers.SerialHandler("/dev/ttyGX2200E", 38400, nil, ch_log, "!AI")
		twg.Add(1)
		go func() {
			defer twg.Done()
			handlers.TeeHandler(logFile, int64(logMaxSize), logArchDir, ch_log, ch_S2N)
			close(ch_S2N)
		}()
	}

	if brokerURL != "" {
		handlers.MQTTHandler(brokerURL, subtopic, pubtopic, ch_S2N, ch_N2S)
	} else if udpbroadcast != "" {
		ch_o1 := make(chan string, 3)
		ch_o2 := make(chan string, 3)
		go func() {
			for msg := range ch_S2N {
				slog.Debug("Dispatch writing", "payload", msg)
				ch_o1 <- msg
				ch_o2 <- msg
			}
		}()
		go handlers.UDPHandler(udpbroadcast, ch_o1)
		handlers.TCPHandler(tcpserver, ch_o2)
	} else {
		handlers.TCPHandler(tcpserver, ch_S2N)
	}

	twg.Wait()
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
