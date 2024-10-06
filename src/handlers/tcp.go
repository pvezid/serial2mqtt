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
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const maxConn = 10

// test avec:
// nc {host} {port}

func TCPHandler(connstr string) chan string {
	if connstr == "" {
		return nil
	}

	ich := make(chan string, 4)

	go func() {
		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

		s, err := newServer(connstr)
		if err != nil {
			slog.Error("TCP Server", "connstr", connstr, "error", err.Error())
			return
		}
		defer s.listener.Close()

		slog.Info("TCP Server is listening", "conn", connstr)

		go s.handleRequests(ich)
		go s.acceptConnections()

		<-sig
		slog.Info("TCP stop signal received")
	}()

	return ich
}

type server struct {
	listener net.Listener
	muConn   sync.Mutex
	connList []net.Conn
}

func newServer(connstr string) (*server, error) {
	listener, err := net.Listen("tcp", connstr)
	if err != nil {
		slog.Error("TCP listen", "connection", connstr, "Error", err)
		return nil, err
	}

	return &server{
		listener: listener,
		connList: make([]net.Conn, 0, maxConn),
	}, nil
}

func (s *server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			slog.Error("TCP accept", "Error", err)
			return
		}
		err = s.addConnection(conn)
		if err != nil {
			slog.Info("TCP", "remote", conn.RemoteAddr(), "error", err)
			conn.Close()
		} else {
			slog.Info("TCP new connection", "remote", conn.RemoteAddr())
		}
	}
}

func (s *server) addConnection(conn net.Conn) error {
	time.Sleep(500 * time.Millisecond)
	s.muConn.Lock()
	defer s.muConn.Unlock()

	if len(s.connList) < maxConn {
		s.connList = append(s.connList, conn)
	} else {
		return errors.New("connection rejected (max number of connections reached)")
	}
	return nil
}

func (s *server) handleRequests(ich <-chan string) {
	slog.Info("TCPOutput init")
	for m := range ich {
		slog.Debug("TCPOutput", "payload", m)
		buff := fmt.Sprintf("%s\r\n", m)
		s.broadcast(buff)
	}
	slog.Info("TCPOutput done")
}

func (s *server) broadcast(buff string) {
	newList := make([]net.Conn, 0, maxConn)
	changed := false
	for _, conn := range s.connList {
		n, err := conn.Write([]byte(buff))
		if err != nil {
			slog.Error("TCPOutput write", "error", err)
			conn.Close()
			changed = true
		} else {
			slog.Debug("TCPOutput wrote", "bytes", n)
			newList = append(newList, conn)
		}
	}
	if changed {
		s.muConn.Lock()
		s.connList = newList
		s.muConn.Unlock()
	}
}
