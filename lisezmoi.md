# Messager

```
sudo useradd -s /usr/sbin/nologin -M -G dialout messager
sudo cat > /etc/systemd/system/messager.service <<EOF
[Unit]
Description=Serial to MQTT messager service
After=network-online.target

[Service]
Type=exec
User=messager
Group=messager
WorkingDirectory=/
KillSignal=SIGTERM
ExecStart=/usr/local/bin/messager -h tcp://domos:1883 -p ser1/tx -s ser1/rx -d /dev/ttyACM0 -b 115200 -l /tmp/serial.log

[Install]
WantedBy=multi-user.target
EOF
sudo chmod 755 /etc/systemd/system/messager.service
sudo systemctl daemon-reload
sudo systemctl enable messager
sudo systemctl start messager
sudo journalctl -f -u messager
```
