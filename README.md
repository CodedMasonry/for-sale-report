# for-sale-report

An app to send an email about recent home listings

For SMTP, if using google, create an APP PASSWORD for it to work
Host: smtp.gmail.com
Port: 587

To setup on linux:

- Build the package
  `go build .`
- Move the built binary to /bin
  `/bin/for-sale-report`
- Move .service and .timer files
  `/etc/systemd/system/`
- Run the following commands to start it
  ```
  sudo systemctl daemon-reload
  sudo systemctl enable --now for-sale-report.timer
  ```
- To confirm it is running
  ```
  systemctl status for-sale-report.timer
  systemctl list-timers
  ```
