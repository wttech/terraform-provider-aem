package instance

import _ "embed"

//go:embed config.yml
var ConfigYML string

//go:embed systemd.conf
var ServiceConf string

//go:embed bin.sh
var BinScript string

//go:embed create.sh
var CreateScript string

//go:embed launch.sh
var LaunchScript string

//go:embed delete.sh
var DeleteScript string
