package instance

import _ "embed"

//go:embed description.md
var DescriptionMD string

//go:embed aem.yml
var ConfigYML string

//go:embed systemd.conf
var ServiceConf string

var CreateScriptInline = []string{
	`sh aemw instance init`,
	`sh aemw instance create`,
}

var LaunchScriptInline = []string{
	`sh aemw osgi config save --pid 'org.apache.sling.jcr.davex.impl.servlets.SlingDavExServlet' --input-string 'alias: /crx/server'`,
	`sh aemw repl agent setup -A --location 'author' --name 'publish' --input-string '{enabled: true, transportUri: "http://localhost:4503/bin/receive?sling:authRequestLogin=1", transportUser: admin, transportPassword: admin, userId: admin}'`,
}

var DeleteScriptInline = []string{
	`sh aemw instance delete`,
}
