[Unit]
Description=Janus Server
After=consul.service
Wants=consul.service

[Service]
ExecStart=/usr/local/bin/janus server
ExecReload=/bin/kill -s HUP $MAINPID
User=${user}
WorkingDirectory=/home/${user}


# restart the consul process if it exits prematurely
Restart=on-failure
StartLimitBurst=3
StartLimitInterval=60s

[Install]
WantedBy=multi-user.target

