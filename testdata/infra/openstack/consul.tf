# Janus standard install
data "template_file" "consul-server-config" {
  count    = "${var.consul_server_instances}"
  template = "${file("../config/consul-server.config.json.tpl")}"

  vars {
    ip_address     = "${element(openstack_compute_instance_v2.consul-server.*.network.0.fixed_ip_v4, count.index)}"
    server_number  = "${var.consul_server_instances}"
    consul_servers = "${jsonencode(openstack_compute_instance_v2.consul-server.*.network.0.fixed_ip_v4)}"
    statsd_ip      = "${openstack_compute_instance_v2.janus-monitoring-server.network.0.fixed_ip_v4}"
    consul_ui      = "false"
  }
}

resource "openstack_compute_instance_v2" "consul-server" {
  count           = "${var.consul_server_instances}"
  region          = "${var.region}"
  name            = "${var.prefix}consul-server-${count.index}"
  image_id        = "${var.janus_compute_image_id}"
  flavor_id       = "${var.janus_compute_flavor_id}"
  key_pair        = "${openstack_compute_keypair_v2.janus.name}"
  security_groups = ["${openstack_compute_secgroup_v2.janus-admin-secgroup.name}"]

  availability_zone = "${var.janus_compute_manager_availability_zone}"

  network {
    uuid = "${openstack_networking_network_v2.janus-admin-net.id}"
  }
}

resource "null_resource" "consul-server-provisioning" {
  count = "${var.consul_server_instances}"

  connection {     
    agent       = false
    # Use janus server as bastion
    bastion_host = "${openstack_compute_floatingip_associate_v2.janus-server-fip.0.floating_ip}"

    user        = "${var.ssh_manager_user}"
    host        = "${element(openstack_compute_instance_v2.consul-server.*.network.0.fixed_ip_v4, count.index)}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    source      = "../config/consul.service"
    destination = "/tmp/consul.service"
  }

  provisioner "file" {
    content     = "${data.template_file.consul-server-config.*.rendered[count.index]}"
    destination = "/tmp/consul-server.config.json"
  }

  provisioner "remote-exec" {
    script = "../scripts/install_consul.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mv /tmp/consul-server.config.json /etc/consul.d/consul-server.config.json",
      "sudo chown root:root /etc/consul.d/*",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable consul.service",
      "sudo systemctl start consul.service",
    ]
  }
}
