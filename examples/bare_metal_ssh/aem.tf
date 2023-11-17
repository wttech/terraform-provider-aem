resource "aem_instance" "single" {
  client {
    type = "ssh"
    settings = {
      host   = "x.x.x.x"
      port   = 22
      user   = "root"
      secure = false
    }
    credentials = {
      private_key = file("private_key.pem")
    }
  }

  files = {
    "lib" = "/data/aemc/aem/home/lib"
  }

  system {}
  compose {}
}

output "aem_instances" {
  value = aem_instance.single.instances
}
