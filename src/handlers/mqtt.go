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
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log/slog"
	"time"
)

func MQTTHandler(brokerURL string, subtopic string, pubtopic string) (chan string, chan string) {
	if brokerURL == "" {
		return nil, nil
	}

	ich := make(chan string, 4)
	och := make(chan string, 4)

	var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
		buff := fmt.Sprintf("%s", msg.Payload())
		slog.Debug("Message received", "topic", msg.Topic(), "payload", buff)
		och <- buff
	}

	go func() {
		opts := mqtt.NewClientOptions()
		opts.AddBroker(brokerURL)
		opts.SetDefaultPublishHandler(messagePubHandler)
		opts.SetConnectionLostHandler(func(client mqtt.Client, reason error) {
			slog.Warn("MQTT connection lost", "broker", brokerURL, "reason", reason.Error())
		})
		opts.SetAutoReconnect(true)
		opts.SetOrderMatters(false)
		opts.SetKeepAlive(25 * time.Second)

		mqttcli := mqtt.NewClient(opts)
		if token := mqttcli.Connect(); token.Wait() && token.Error() != nil {
			slog.Error("MQTT connect", "broker", brokerURL, "error", token.Error())
			return
		}

		if subtopic != "" {
			if token := mqttcli.Subscribe(subtopic, 1, nil); token.Wait() && token.Error() != nil {
				slog.Error("MQTT subscribe", "topic", subtopic, "error", token.Error())
			} else {
				slog.Info("Subscribed", "topic", subtopic)
			}
		}
		for msg := range ich {
			if pubtopic != "" {
				if token := mqttcli.Publish(pubtopic, 0, false, msg); token.Wait() && token.Error() != nil {
					slog.Error("MQTT publish", "topic", pubtopic, "error", token.Error())
				} else {
					slog.Debug("Message published", "topic", pubtopic, "payload", msg)
				}
			}
		}
	}()

	return ich, och
}
