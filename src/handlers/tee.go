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
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func TeeHandler(fileName string, chunkSize int64, logArchDir string, ich <-chan string, och chan<- string) {
	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	archDir := isWriteableDirectory(logArchDir)
	fo := createFile(fileName)
loop:
	for {
		select {
		case msg := <-ich:
			och <- msg
			if fo != nil {
				if info, err := fo.Stat(); err == nil {
					if info.Size() > chunkSize {
						fo.Close()
						if archDir {
							name := fo.Name()
							slog.Info("Tee archiving", "name", name, "arch directory", logArchDir)
							go moveFile(name, path.Join(logArchDir, path.Base(name)))
						}
						fo = createFile(fileName)
					}
				}
				buff := fmt.Sprintln(msg)
				fo.Write([]byte(buff))
			}
		case <-sig:
			slog.Info("Tee stop signal received")
			break loop
		}
	}
	fo.Close()
	if archDir {
		name := fo.Name()
		slog.Info("Tee archiving", "name", name, "arch directory", logArchDir)
		moveFile(name, path.Join(logArchDir, path.Base(name)))
	}
}

func createFile(fileName string) *os.File {
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	ts := fmt.Sprintf("%s", time.Now().Format(time.RFC3339))
	fn := fmt.Sprintf("%s-%s%s", base, ts, ext)
	fo, err := os.Create(fn)
	if err != nil {
		slog.Error("Tee create file", "name", fn, "error", err)
	} else {
		slog.Info("Tee file created", "name", fn, "ts", ts)
	}
	return fo
}

func isWriteableDirectory(name string) bool {
	fi, err := os.Stat(name)
	if err != nil {
		slog.Error("Tee directory", "name", name, "error", err)
		return false
	}
	mode := fi.Mode()
	if mode.IsDir() {
		file, err := os.CreateTemp(name, "tmpfile")
		if err != nil {
			slog.Error("Tee directory", "name", name, "error", err)
			return false
		}
		file.Close()
		os.Remove(file.Name())
		slog.Error("Tee directory is writeable", "name", name)
		return true
	}
	slog.Error("Tee not a directory", "name", name)
	return false
}

func moveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}

	outputFile, err := os.Create(destPath)
	if err != nil {
		inputFile.Close()
		return err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		inputFile.Close()
		return err
	}

	inputFile.Close() // for Windows, close before trying to remove: https://stackoverflow.com/a/64943554/246801

	err = os.Remove(sourcePath)
	if err != nil {
		return err
	}
	return nil
}
