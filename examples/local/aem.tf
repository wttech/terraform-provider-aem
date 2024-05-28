resource "aem_instance" "single" {
  client {
    type     = "local"
    settings = {
    }
  }

  system {
    data_dir  = local.compose_dir
    work_dir  = local.work_dir
    bootstrap = {
      inline = [
      ]
    }
  }

  compose {
    config = file("aem.yml")
    create = {
      inline = [
        "mkdir -p ${local.compose_dir}/aem/home/lib",
        "cp ${local.library_dir}/* ${local.compose_dir}/aem/home/lib",
        "sh aemw instance init",
        "sh aemw instance create",
      ]
    }
    configure = {
      inline = [
        "sh aemw osgi config save --pid 'org.apache.sling.jcr.davex.impl.servlets.SlingDavExServlet' --input-string 'alias: /crx/server'",
        "sh aemw repl agent setup -A --location 'author' --name 'publish' --input-string '{enabled: true, transportUri: \"http://localhost:4503/bin/receive?sling:authRequestLogin=1\", transportUser: admin, transportPassword: admin, userId: admin}'",
      ]
    }
  }
}

locals {
  env         = "local"
  data_dir    = "~/data/${local.env}"
  compose_dir = "${local.data_dir}/aemc"
  work_dir    = "~/tmp/${local.env}/aemc"
  library_dir = "~/lib"
}

output "aem_instances" {
  value = "127.0.0.1"
}
