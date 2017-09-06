{
      "advertise_addr": "${ip_address}",
      "client_addr": "0.0.0.0",
      "data_dir": "/var/consul",
      "server": true,
      "bootstrap_expect": ${server_number},
      "retry_join": ${consul_servers},
      "telemetry": {
            "statsd_address": "${statsd_ip}:8125"
      }
}