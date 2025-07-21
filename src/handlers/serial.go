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

package handlers

import (
	"bufio"
	"fmt"
	"go.bug.st/serial"
	"log/slog"
	"strings"
	"sync"
	"time"
)

func SerialPortList() {
	ports, err := serial.GetPortsList()
	if err != nil {
		slog.Error("GetPortsList", "error", err)
		return
	}
	if len(ports) == 0 {
		slog.Info("No serial ports found!")
	}
	for _, port := range ports {
		slog.Info("Found port", "port", port)
	}
}

func SerialHandler(dev string, baud int, filter string, raw bool) (chan string, chan string) {
	if dev == "" {
		return nil, nil
	}

	ich := make(chan string, 4)
	och := make(chan string, 4)

	go func() {
		mode := &serial.Mode{
			BaudRate: baud,
		}
		for {
			slog.Info("Serial open", "port", dev, "baud", baud)
			serialport, err := serial.Open(dev, mode)
			if err != nil {
				slog.Error("Serial", "port", dev, "error", err)
				time.Sleep(18 * time.Second)
				continue
			}

			// on introduit un control channel pour pouvoir terminer serialOutput
			// sans devoir fermer le channel "appelant" ich
			// serialOutput est non bloquant
			ctrl_ch, wg := serialOutput(serialport, ich)

			// serialInput est bloquant
			serialInput(dev, serialport, filter, raw, och)
			close(ctrl_ch) // force l'arrêt de serialOutput
			wg.Wait()      // on attend la fin de la goroutine de serialOutput

			serialport.Close()
			slog.Info("Serial closed")
			time.Sleep(5 * time.Second)
		}
	}()

	return ich, och
}

func serialInput(dev string, serialport serial.Port, filter string, raw bool, och chan<- string) {
	slog.Info("SerialInput init")
	scanner := bufio.NewScanner(serialport)
	warmup := 2
	for scanner.Scan() {
		msg := scanner.Text()
		if warmup == 0 {
			if filter == "" || strings.HasPrefix(msg, filter) {
				slog.Debug("SerialInput read", "port", dev, "payload", msg)
				if raw {
					och <- fmt.Sprintf("%s", msg)
				} else {
					och <- fmt.Sprintf("%v:%s:%s", time.Now().UnixNano(), dev, msg)
				}
			}
		} else {
			warmup -= 1
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Error("SerialInput scanner", "error", err)
	}
	slog.Info("SerialInput done")
}

func serialOutput(serialport serial.Port, ich <-chan string) (chan struct{}, *sync.WaitGroup) {
	ctrl_ch := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		slog.Info("SerialOutput init")
	Loop:
		for {
			select {
			case m := <-ich:
				buff := fmt.Sprintln(m)
				_, err := serialport.Write([]byte(buff))
				if err != nil {
					slog.Error("SerialOutput write", "error", err)
				} else {
					slog.Debug("SerialOutput wrote", "payload", m)
				}
			case _, ok := <-ctrl_ch:
				if !ok {
					break Loop
				}
			}
		}
		slog.Info("SerialOutput done")
	}()

	return ctrl_ch, &wg
}
