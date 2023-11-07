package instance

import _ "embed"

//go:embed config.yml
var ConfigYML string

//go:embed systemd.conf.go.tpl
var ServiceTemplate string
