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
	"sync"
	"time"
)

type SerialPort struct {
	serial.Port
	Name string
}

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

func SerialHandler(dev string, baud int) (chan string, chan string) {
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
			serial := &SerialPort{serialport, dev}

			// on introduit un control channel pour pouvoir terminer serialOutput
			// sans devoir fermer le channel "appelant" ich
			// serialOutput est non bloquant
			ctrl_ch, wg := serial.serialOutput(ich)

			// serialInput est bloquant
			serial.serialInput(och)
			close(ctrl_ch) // force l'arrêt de serialOutput
			wg.Wait()      // on attend la fin de la goroutine de serialOutput

			serial.Close()
			slog.Info("Serial closed", "port", dev)
			time.Sleep(5 * time.Second)
		}
	}()

	return ich, och
}

func (serialport *SerialPort) serialInput(och chan<- string) {
	slog.Info("SerialInput init", "port", serialport.Name)
	scanner := bufio.NewScanner(serialport)
	warmup := 2
	for scanner.Scan() {
		msg := scanner.Text()
		if warmup == 0 {
			slog.Debug("SerialInput read", "port", serialport.Name, "payload", msg)
			och <- fmt.Sprintf("%s", msg)
		} else {
			warmup -= 1
		}
	}
	if err := scanner.Err(); err != nil {
		slog.Error("SerialInput scanner", "port", serialport.Name, "error", err)
	}
	slog.Info("SerialInput done", "port", serialport.Name)
}

func (serialport *SerialPort) serialOutput(ich <-chan string) (chan struct{}, *sync.WaitGroup) {
	ctrl_ch := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		slog.Info("SerialOutput init", "port", serialport.Name)
	Loop:
		for {
			select {
			case m := <-ich:
				buff := fmt.Sprintln(m)
				_, err := serialport.Write([]byte(buff))
				if err != nil {
					slog.Error("SerialOutput write", "port", serialport.Name, "error", err)
				} else {
					slog.Debug("SerialOutput wrote", "port", serialport.Name, "payload", m)
				}
			case <-ctrl_ch:
				break Loop
			}
		}
		slog.Info("SerialOutput done", "port", serialport.Name)
	}()

	return ctrl_ch, &wg
}
