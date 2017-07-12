resource "openstack_networking_floatingip_v2" "janus-admin-fp-janus" {
  count  = "${var.janus_instances}"
  region = "${var.region}"
  pool   = "${var.public_network_name}"

  depends_on = [
    "openstack_networking_router_interface_v2.janus-admin-router-port",
  ]
}

# Janus standard install
data "template_file" "janus-server-config" {
  count    = "${var.janus_instances}"
  template = "${file("../config/janus.config.json.tpl")}"

  vars {
    ks_url      = "${var.keystone_url}"
    ks_tenant   = "${var.keystone_tenant}"
    ks_user     = "${var.keystone_user}"
    ks_password = "${var.keystone_password}"
    region      = "${var.region}"
    priv_net    = "${openstack_networking_network_v2.janus-admin-net.name}"
    prefix      = "${var.prefix}janus"
    secgrp      = "${openstack_compute_secgroup_v2.janus-admin-secgroup.name}"
    statsd_ip   = "${openstack_compute_instance_v2.janus-monitoring-server.network.0.fixed_ip_v4}"
  }
}

data "template_file" "janus-server-service" {
  count    = "${var.janus_instances}"
  template = "${file("../config/janus.service.tpl")}"

  vars {
    user = "${var.ssh_manager_user}"
  }
}

data "template_file" "janus-consul-checks" {
  template = "${file("../config/janus-consul-check.json.tpl")}"

  vars {
    janus_id = "${count.index}"
    janus_ip = "${element(openstack_compute_floatingip_associate_v2.janus-server-fip.*.floating_ip, count.index)}"
  }
}

data "template_file" "consul-agent-config" {
  count    = "${var.janus_instances}"
  template = "${file("../config/consul-agent.config.json.tpl")}"

  vars {
    ip_address     = "${element(openstack_compute_instance_v2.janus-server.*.network.0.fixed_ip_v4, count.index)}"
    consul_servers = "${jsonencode(join(",", openstack_compute_instance_v2.consul-server.*.network.0.fixed_ip_v4))}"
    statsd_ip      = "${openstack_compute_instance_v2.janus-monitoring-server.network.0.fixed_ip_v4}"
  }
}

resource "openstack_compute_instance_v2" "janus-server" {
  count           = "${var.janus_instances}"
  region          = "${var.region}"
  name            = "${var.prefix}janus-server"
  image_id        = "${var.janus_compute_image_id}"
  flavor_id       = "${var.janus_compute_flavor_id}"
  key_pair        = "${openstack_compute_keypair_v2.janus.name}"
  security_groups = ["${openstack_compute_secgroup_v2.janus-admin-secgroup.name}"]

  availability_zone = "${var.janus_compute_manager_availability_zone}"

  network {
    uuid = "${openstack_networking_network_v2.janus-admin-net.id}"
  }
}

resource "openstack_compute_floatingip_associate_v2" "janus-server-fip" {
  count       = "${var.janus_instances}"
  floating_ip = "${element(openstack_networking_floatingip_v2.janus-admin-fp-janus.*.address, count.index)}"
  instance_id = "${element(openstack_compute_instance_v2.janus-server.*.id, count.index)}"
}

resource "null_resource" "janus-server-provisioning" {
  count = "${var.janus_instances}"

  connection {
    user        = "${var.ssh_manager_user}"
    host        = "${element(openstack_compute_floatingip_associate_v2.janus-server-fip.*.floating_ip, count.index)}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    source      = "${var.ssh_key_file}"
    destination = "${var.ssh_key_file}"
  }

  provisioner "file" {
    source      = "../janus"
    destination = "/tmp/janus"
  }

  provisioner "file" {
    source      = "../config/consul.service"
    destination = "/tmp/consul.service"
  }

  provisioner "file" {
    content     = "${data.template_file.janus-server-config.*.rendered[count.index]}"
    destination = "/tmp/config.janus.json"
  }

  provisioner "file" {
    content     = "${data.template_file.consul-agent-config.*.rendered[count.index]}"
    destination = "/tmp/consul-agent.config.json"
  }

  provisioner "file" {
    content     = "${data.template_file.janus-consul-checks.*.rendered[count.index]}"
    destination = "/tmp/janus-consul-check.json"
  }

  provisioner "file" {
    content     = "${data.template_file.janus-server-service.rendered}"
    destination = "/tmp/janus.service"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mkdir -p /etc/consul.d",
      "sudo mv /tmp/janus-consul-check.json /etc/consul.d/",
      "sudo mv /tmp/consul-agent.config.json /etc/consul.d/",
      "sudo chown root:root /etc/consul.d/*",
      "chmod 400 ${var.ssh_key_file}",
      "sudo mv /tmp/janus /usr/local/bin",
      "sudo chmod +x /usr/local/bin/janus",
      "sudo chown root:root /usr/local/bin/janus",
      "sudo mv /tmp/consul.service /etc/systemd/system/consul.service",
      "sudo chown root:root /etc/systemd/system/consul.service",
      "sudo mv /tmp/janus.service /etc/systemd/system/janus.service",
      "sudo chown root:root /etc/systemd/system/janus.service",
      "sudo yum install -q -y unzip zip python2-pip wget",
      "cd /tmp && wget -q https://releases.hashicorp.com/consul/0.8.1/consul_0.8.1_linux_amd64.zip && sudo unzip /tmp/consul_0.8.1_linux_amd64.zip -d /usr/local/bin",
      "cd /tmp && wget -q https://releases.hashicorp.com/terraform/0.9.11/terraform_0.9.11_linux_amd64.zip && sudo unzip /tmp/terraform_0.9.11_linux_amd64.zip -d /usr/local/bin",
      "sudo -H pip install -q pip --upgrade",
      "sudo -H pip install -q ansible==2.3.1.0",
      "mv /tmp/config.janus.json ~/config.janus.json",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable consul.service janus.service",
      "sudo systemctl start consul.service janus.service",
    ]
  }
}

output "janus_addresses" {
  value = ["${openstack_compute_floatingip_associate_v2.janus-server-fip.*.floating_ip}"]
}
