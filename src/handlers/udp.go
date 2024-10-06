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
	"fmt"
	"log/slog"
	"net"
)

// test avec:
// nc -kluw 0 {port}

func UDPHandler(connstr string, ich <-chan string) {

	addr, err := net.ResolveUDPAddr("udp4", connstr)
	if err != nil {
		slog.Error("UDP ResolveUDPAddr", "Error", err)
		return
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		slog.Error("UDP DialUDP", "Error", err)
		return
	}
	defer conn.Close()

	slog.Info("UDP is ready to send", "conn", connstr)

	slog.Info("UDPOutput init")
	for m := range ich {
		buff := fmt.Sprintln(m)
		_, err := conn.Write([]byte(buff))
		if err != nil {
			slog.Error("UDPOutput write", "error", err)
		} else {
			slog.Debug("UDPOutput wrote", "payload", m)
		}
	}
	slog.Info("UDPOutput done")
}
