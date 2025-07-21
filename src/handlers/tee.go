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
	"path"
	"path/filepath"
	"strings"
	"time"
)

type MaskableMessage struct {
	Msg    string
	Device string
	Masked bool
}

type logger struct {
	tmplName         string
	fileName         string
	chunkSize        int64
	logArchDir       string
	archDirWriteable bool
	fp               *os.File
}

func TeeHandler(fileName string, chunkSize int64, logArchDir string) (chan MaskableMessage, chan string) {
	ich := make(chan MaskableMessage, 4)
	och := make(chan string, 4)

	go func() {
		log := &logger{tmplName: fileName, chunkSize: chunkSize, logArchDir: logArchDir}
		log.createFile()

		for msg := range ich {
			slog.Debug("Tee", "payload", msg)
			var s string
			if msg.Masked {
				s = fmt.Sprintf("%v:#%s:%s", time.Now().UnixNano(), msg.Device, msg.Msg)
			} else {
				s = fmt.Sprintf("%v:%s:%s", time.Now().UnixNano(), msg.Device, msg.Msg)
				och <- msg.Msg
			}
			log.writeFile(s)
		}

		log.closeFile()
		close(och)
	}()

	return ich, och
}

func (log *logger) createFile() {
	if log.tmplName == "" {
		return
	}
	log.archDirWriteable = isWriteableDirectory(log.logArchDir)
	ext := filepath.Ext(log.tmplName)
	base := strings.TrimSuffix(log.tmplName, ext)
	ts := fmt.Sprintf("%s", time.Now().Format(time.RFC3339))
	log.fileName = fmt.Sprintf("%s-%s%s", base, ts, ext)
	var err error
	log.fp, err = os.Create(log.fileName)
	if err != nil {
		slog.Error("Tee create file", "name", log.fileName, "error", err)
	} else {
		slog.Info("Tee file created", "name", log.fileName)
	}
}

func (log *logger) writeFile(msg string) {
	if log.fp != nil {
		if info, err := log.fp.Stat(); err == nil {
			if info.Size() > log.chunkSize {
				log.fp.Close()
				if log.archDirWriteable {
					slog.Info("Tee archiving", "name", log.fileName, "arch directory", log.logArchDir)
					go moveFile(log.fileName, path.Join(log.logArchDir, path.Base(log.fileName)))
				}
				log.createFile()
			}
		}
		buff := fmt.Sprintln(msg)
		log.fp.Write([]byte(buff))
	}
}

func (log *logger) closeFile() {
	if log.fp != nil {
		log.fp.Close()
		if log.archDirWriteable {
			slog.Info("Tee archiving", "name", log.fileName, "arch directory", log.logArchDir)
			moveFile(log.fileName, path.Join(log.logArchDir, path.Base(log.fileName)))
		}
	}
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
