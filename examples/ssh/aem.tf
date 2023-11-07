resource "aem_instance" "single" {
  depends_on = [aws_instance.aem_single, aws_volume_attachment.aem_single_data]

  client {
    type     = "ssh"
    settings = {
      host             = aws_instance.aem_single.public_ip
      port             = 22
      user             = local.ssh_user
      private_key_file = local.ssh_private_key # cannot be put into state as this is OS-dependent
    }
  }

  system {
    data_dir  = local.aem_single_compose_dir
    bootstrap = <<SHELL
      #!/bin/sh
      (
        echo "Mounting EBS volume into data directory"
        sudo mkfs -t ext4 ${local.aem_single_data_device} && \
        sudo mkdir -p ${local.aem_single_data_dir} && \
        sudo mount ${local.aem_single_data_device} ${local.aem_single_data_dir} && \
        sudo chown -R ${local.ssh_user} ${local.aem_single_data_dir} && \
        echo '${local.aem_single_data_device} ${local.aem_single_data_dir} ext4 defaults 0 0' | sudo tee -a /etc/fstab
      ) && (
        echo "Copying AEM library files"
        sudo yum install -y unzip && \
        curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
        unzip -q awscliv2.zip && \
        sudo ./aws/install --update && \
        mkdir -p "${local.aem_single_compose_dir}/aem/home/lib" && \
        aws s3 cp --recursive --no-progress "s3://aemc/instance/classic/" "${local.aem_single_compose_dir}/aem/home/lib"
      )
    SHELL
  }

  compose {
    version = "1.5.8"
    launch = <<SHELL
      #!/bin/sh
      sh aemw osgi bundle install --url "https://github.com/neva-dev/felix-search-webconsole-plugin/releases/download/2.0.0/search-webconsole-plugin-2.0.0.jar" && \
      sh aemw osgi config save --pid "org.apache.sling.jcr.davex.impl.servlets.SlingDavExServlet" --input-string "alias: /crx/server" && \
      echo "
      enabled: true
      transportUri: http://localhost:4503/bin/receive?sling:authRequestLogin=1
      transportUser: admin
      transportPassword: admin
      userId: admin
      " | sh aemw repl agent setup -A --location "author" --name "publish" && \
      sh aemw package deploy --file "aem/home/lib/aem-service-pkg-6.5.*.0.zip"
    SHELL

    config = <<YAML
    # AEM instances to work with
    instance:

      # Full details of local or remote instances
      config:
        local_author:
          active: true
          http_url: http://127.0.0.1:4502
          user: admin
          password: admin
          run_modes: [local]
          jvm_opts:
            - -server
            - -Djava.awt.headless=true
            - -Djava.io.tmpdir=[[canonicalPath .Path "aem/home/tmp"]]
            - -agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=0.0.0.0:14502
            - -Duser.language=en
            - -Duser.country=US
            - -Duser.timezone=UTC
          start_opts: []
          secret_vars:
            - ACME_SECRET=value
          env_vars:
            - ACME_VAR=value
          sling_props: []
        local_publish:
          active: true
          http_url: http://127.0.0.1:4503
          user: admin
          password: admin
          run_modes: [local]
          jvm_opts:
            - -server
            - -Djava.awt.headless=true
            - -Djava.io.tmpdir=[[canonicalPath .Path "aem/home/tmp"]]
            - -agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=0.0.0.0:14503
            - -Duser.language=en
            - -Duser.country=US
            - -Duser.timezone=UTC
          start_opts: []
          secret_vars:
            - ACME_SECRET=value
          env_vars:
            - ACME_VAR=value
          sling_props: []

      # Tuning performance & reliability
      # 'auto'     - for more than 1 local instances - 'serial', otherwise 'parallel'
      # 'parallel' - for working with remote instances
      # 'serial'   - for working with local instances
      processing_mode: auto

      # HTTP client settings
      http:
        timeout: 10m
        debug: false
        disable_warn: true

      # State checking
      check:
        # Time to wait before first state checking (to avoid false-positives)
        warmup: 1s
        # Time to wait for next state checking
        interval: 6s
        # Number of successful check attempts that indicates end of checking
        done_threshold: 3
        # Wait only for those instances whose state has been changed internally (unaware of external changes)
        await_strict: true
        # Max time to wait for the instance to be healthy after executing the start script or e.g deploying a package
        await_started:
          timeout: 30m
        # Max time to wait for the instance to be stopped after executing the stop script
        await_stopped:
          timeout: 10m
        # Max time in which socket connection to instance should be established
        reachable:
          timeout: 3s
        # Bundle state tracking
        bundle_stable:
          symbolic_names_ignored: []
        # OSGi events tracking
        event_stable:
          # Topics indicating that instance is not stable
          topics_unstable:
            - "org/osgi/framework/ServiceEvent/*"
            - "org/osgi/framework/FrameworkEvent/*"
            - "org/osgi/framework/BundleEvent/*"
          # Ignored service names to handle known issues
          details_ignored:
            - "*.*MBean"
            - "org.osgi.service.component.runtime.ServiceComponentRuntime"
            - "java.util.ResourceBundle"
          received_max_age: 5s
        # Sling Installer tracking
        installer:
          # JMX state checking
          state: true
          # Pause Installation nodes checking
          pause: true
        # Specific endpoints / paths (like login page)
        path_ready:
          timeout: 10s


      # Managed locally (set up automatically)
      local:
        # Current runtime dir (Sling launchpad, JCR repository)
        unpack_dir: "aem/home/var/instance"
        # Archived runtime dir (AEM backup files '*.aemb.zst')
        backup_dir: "aem/home/var/backup"

        # Oak Run tool options (offline instance management)
        oak_run:
          download_url: "https://repo1.maven.org/maven2/org/apache/jackrabbit/oak-run/1.44.0/oak-run-1.44.0.jar"
          store_path: "crx-quickstart/repository/segmentstore"

        # Source files
        quickstart:
          # AEM SDK ZIP or JAR
          dist_file: 'aem/home/lib/{aem-sdk,cq-quickstart}-*.{zip,jar}'
          # AEM License properties file
          license_file: "aem/home/lib/license.properties"

      # Status discovery (timezone, AEM version, etc)
      status:
        timeout: 500ms

      # JCR Repository
      repo:
        property_change_ignored:
          # AEM assigns them automatically
          - "jcr:created"
          - "cq:lastModified"
          # AEM encrypts it right after changing by replication agent setup command
          - "transportPassword"

      # CRX Package Manager
      package:
        # Force re-uploading/installing of snapshot AEM packages (just built / unreleased)
        snapshot_patterns: [ "**/*-SNAPSHOT.zip" ]
        # Use checksums to avoid re-deployments when snapshot AEM packages are unchanged
        snapshot_deploy_skipping: true
        # Disable following workflow launchers for a package deployment time only
        toggled_workflows: [/libs/settings/workflow/launcher/config/update_asset_*,/libs/settings/workflow/launcher/config/dam_*]
        # Also sub-packages
        install_recursive: true
        # Use slower HTML endpoint for deployments but with better troubleshooting
        install_html:
          enabled: false
          # Print HTML directly to console instead of writing to file
          console: false
          # Fail on case 'installed with errors'
          strict: true

      # OSGi Framework
      osgi:
        shutdown_delay: 3s

        bundle:
          install:
            start: true
            start_level: 20
            refresh_packages: true

      # Crypto Support
      crypto:
        key_bundle_symbolic_name: com.adobe.granite.crypto.file

      # Workflow Manager
      workflow:
        launcher:
          lib_root: /libs/settings/workflow/launcher
          config_root: /conf/global/settings/workflow/launcher
          toggle_retry:
            timeout: 10m
            delay: 10s

    java:
      # Require following versions before e.g running AEM instances
      version_constraints: ">= 11, < 12"

      # Pre-installed local JDK dir
      # a) keep it empty to download open source Java automatically for current OS and architecture
      # b) set it to absolute path or to env var '[[.Env.JAVA_HOME]]' to indicate where closed source Java like Oracle is installed
      home_dir: ""

      # Auto-installed JDK options
      download:
        # Source URL with template vars support
        url: "https://github.com/adoptium/temurin11-binaries/releases/download/jdk-11.0.18%2B10/OpenJDK11U-jdk_[[.Arch]]_[[.Os]]_hotspot_11.0.18_10.[[.ArchiveExt]]"
        # Map source URL template vars to be compatible with Adoptium Java
        replacements:
          # Var 'Os' (GOOS)
          "darwin": "mac"
          # Var 'Arch' (GOARCH)
          "x86_64": "x64"
          "amd64": "x64"
          "386": "x86-32"
          # enforce non-ARM Java as some AEM features are not working on ARM (e.g Scene7)
          "arm64": "x64"
          "aarch64": "x64"

    base:
      # Location of temporary files (downloaded AEM packages, etc)
      tmp_dir: aem/home/tmp
      # Location of supportive tools (downloaded Java, OakRun, unpacked AEM SDK)
      tool_dir: aem/home/opt

    log:
      level: info
      timestamp_format: "2006-01-02 15:04:05"
      full_timestamp: true

    input:
      format: yml
      file: STDIN

    output:
      format: text
      log:
        # File path of logs written especially when output format is different than 'text'
        file: aem/home/var/log/aem.log
        # Controls where outputs and logs should be written to when format is 'text' (console|file|both)
        mode: console
    YAML
  }
}

locals {
  // https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/device_naming.html#device-name-limits
  aem_single_data_device = "/dev/nvme1n1"
  aem_single_data_dir    = "/data"
  aem_single_compose_dir = "${local.aem_single_data_dir}/aemc"
}

output "aem_instances" {
  value = aem_instance.single.instances
}
