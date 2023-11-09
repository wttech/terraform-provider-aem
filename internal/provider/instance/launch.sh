#!/bin/sh

sh aemw osgi config save --pid "org.apache.sling.jcr.davex.impl.servlets.SlingDavExServlet" --input-string "alias: /crx/server" && \
echo "
enabled: true
transportUri: http://localhost:4503/bin/receive?sling:authRequestLogin=1
transportUser: admin
transportPassword: admin
userId: admin
" | sh aemw repl agent setup -A --location "author" --name "publish" && \
sh aemw package deploy --file "aem/home/lib/aem-service-pkg-6.5.*.0.zip"
