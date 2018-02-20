resource "openstack_compute_instance_v2" "nfs-server" {
  region          = "${var.region}"
  name            = "${var.prefix}nfs-server"
  image_id        = "${var.yorc_compute_image_id}"
  flavor_id       = "${var.yorc_compute_flavor_id}"
  key_pair        = "${openstack_compute_keypair_v2.yorc.name}"
  security_groups = ["${openstack_compute_secgroup_v2.yorc-admin-secgroup.name}"]

  availability_zone = "${var.yorc_compute_manager_availability_zone}"

  network {
    uuid = "${openstack_networking_network_v2.yorc-admin-net.id}"
  }
}

data "template_file" "nfs-consul-checks" {
  template = "${file("../config/nfs-consul-check.json.tpl")}"

  vars {
    ip_address = "${openstack_compute_instance_v2.nfs-server.network.0.fixed_ip_v4}"
  }
}

data "template_file" "nfs-consul-agent-config" {
  template = "${file("../config/consul-agent.config.json.tpl")}"

  vars {
    ip_address     = "${openstack_compute_instance_v2.nfs-server.network.0.fixed_ip_v4}"
    consul_servers = "${jsonencode(openstack_compute_instance_v2.consul-server.*.network.0.fixed_ip_v4)}"
    statsd_ip      = "${openstack_compute_instance_v2.yorc-monitoring-server.network.0.fixed_ip_v4}"
    consul_ui      = "false"
  }
}

data "template_file" "nfs-exports" {
  template = "${file("../config/nfs-exports.tpl")}"

  vars {
    nfs-exports = "${join(" ", formatlist("%s(rw,sync,no_root_squash)", openstack_compute_instance_v2.yorc-server.*.network.0.fixed_ip_v4))}"
  }
}

resource "null_resource" "nfs-server-provisioning" {
  connection {
    # Use yorc server as bastion
    bastion_host = "${openstack_compute_floatingip_associate_v2.yorc-server-fip.0.floating_ip}"

    user        = "${var.ssh_manager_user}"
    host        = "${element(openstack_compute_instance_v2.nfs-server.*.network.0.fixed_ip_v4, count.index)}"
    private_key = "${file("${var.ssh_key_file}")}"
  }

  provisioner "file" {
    source      = "../config/consul.service"
    destination = "/tmp/consul.service"
  }

  provisioner "file" {
    content     = "${data.template_file.nfs-consul-agent-config.rendered}"
    destination = "/tmp/consul-agent.config.json"
  }

  provisioner "file" {
    content     = "${data.template_file.nfs-consul-checks.rendered}"
    destination = "/tmp/nfs-consul-check.json"
  }

  provisioner "file" {
    content     = "${data.template_file.nfs-exports.rendered}"
    destination = "/tmp/nfs-exports"
  }

  provisioner "remote-exec" {
    script = "../scripts/install_consul.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "sudo mv /tmp/consul-agent.config.json /etc/consul.d/",
      "sudo mv /tmp/nfs-consul-check.json /etc/consul.d/",
      "sudo chown root:root /etc/consul.d/*",
      "sudo yum install -y -q nfs-utils",
      "sudo systemctl daemon-reload",
      "sudo systemctl enable consul.service nfs-server.service",
      "sudo systemctl start consul.service nfs-server.service",
      "sudo mkdir -p /mountedStorageNFS/yorc-server/work",
      "sudo chown ${var.ssh_manager_user}:${var.ssh_manager_user} /mountedStorageNFS/yorc-server/work",
      "sudo mv /tmp/nfs-exports /etc/exports && sudo chown root:root /etc/exports",
      "sudo exportfs -r",
    ]
  }
}
