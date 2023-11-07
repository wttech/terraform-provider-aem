[Unit]
Description=AEM Instances
Requires=network.target
After=cloud-final.service

[Service]
Type=forking

ExecStart=su - "{{.USER}}" -c ". /etc/profile && cd {{.DATA_DIR}} && sh aemw instance start"
ExecStop=su - "{{.USER}}" -c ". /etc/profile && cd {{.DATA_DIR}} && sh aemw instance stop"
ExecReload=su - "{{.USER}}" -c ". /etc/profile && cd {{.DATA_DIR}} && sh aemw instance restart"
KillMode=process
RemainAfterExit=yes
TimeoutStartSec=1810
TimeoutStopSec=190
LimitNOFILE=20000

[Install]
WantedBy=cloud-init.target
