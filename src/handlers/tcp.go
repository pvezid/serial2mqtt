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

type client chan<- string // an outgoing message channel

// test avec:
// nc {host} {port}

func TCPHandler(connstr string) chan<- string {
	if connstr == "" {
		return nil
	}

	ich := make(chan string, 4)

	go func() {
		listener, err := net.Listen("tcp", connstr)
		if err != nil {
			slog.Error("TCP listen", "connection", connstr, "error", err)
			return
		}
		defer listener.Close()

		slog.Info("TCP Server is listening", "connection", connstr)

		entering := make(chan client)
		leaving := make(chan client)

		go broadcaster(ich, entering, leaving)

		for {
			conn, err := listener.Accept()
			if err != nil {
				slog.Error("TCP accept", "error", err)
				continue
			}
			go handleConnection(conn, entering, leaving)
		}
	}()

	return ich
}

func broadcaster(ich <-chan string, entering chan client, leaving chan client) {
	clients := make(map[client]bool) // all connected clients
loop:
	for {
		select {
		case msg, ok := <-ich:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			if ok {
				for cli := range clients {
					cli <- msg
				}
			} else {
				break loop
			}

		case cli := <-entering:
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
	for cli := range clients {
		close(cli)
	}
}

func handleConnection(conn net.Conn, entering chan<- client, leaving chan<- client) {
	slog.Info("TCP new connection", "remote", conn.RemoteAddr().String())
	ch := make(chan string) // outgoing client messages
	entering <- ch
	for msg := range ch {
		buff := fmt.Sprintf("%s\r\n", msg) // terminaison avec CRLF
		_, err := conn.Write([]byte(buff))
		if err != nil {
			leaving <- ch
			break
		}
	}
	slog.Info("TCP closing connection", "remote", conn.RemoteAddr().String())
	conn.Close()
}
