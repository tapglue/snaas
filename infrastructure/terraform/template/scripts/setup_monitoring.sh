#!/bin/bash

set -ex

# Base setup
sudo apt-get update
sudo apt-get install -y curl

# Install Prometheus
echo -n 'deb http://deb.robustperception.io/ precise nightly' | sudo tee /etc/apt/sources.list.d/robustperception.io.list > /dev/null
curl https://s3-eu-west-1.amazonaws.com/deb.robustperception.io/41EFC99D.gpg | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y alertmanager node-exporter pushgateway prometheus

# Setup and start Prometheus
sudo service prometheus start
sudo update-rc.d prometheus defaults 95 10

# Install Grafana
echo -n 'deb https://packagecloud.io/grafana/stable/debian/ jessie main' | sudo tee /etc/apt/sources.list.d/packagecloud.io.list > /dev/null
curl https://packagecloud.io/gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install -y grafana

# Setup and start Grafana
sudo /bin/systemctl daemon-reload
sudo /bin/systemctl enable grafana-server
sudo /bin/systemctl start grafana-server

sleep 30

# Setup Grafana dashboard
sudo mkdir -p /var/lib/grafana/dashboards
echo '${dashboard}' | sudo tee /var/lib/grafana/dashboards/dashboard-ops.json > /dev/null

# Add Prometheus data source
curl -vvv \
  -X POST \
  -u admin:admin \
  -H 'Content-Type: application/json;charset=UTF-8' \
  --data-binary '{"name":"prometheus", "type":"prometheus","url":"http://localhost:9090","access":"proxy","isDefault":true}' \
  'http://0.0.0.0:3000/api/datasources'

# Setup Grafana config
PASSWORD=$(date +%s | sha256sum | base64 | head -c 32 ; echo)

echo "
[auth]
disable_login_form = true
[auth.basic]
enabled = false
[auth.google]
enabled = true
client_id = ${google_client_id}
client_secret = ${google_client_secret}
scopes = https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/userinfo.email
auth_url = https://accounts.google.com/o/oauth2/auth
token_url = https://accounts.google.com/o/oauth2/token
allowed_domains = ${domain} ${domain_canonical}
allow_sign_up = true
[dashboards.json]
enabled = true
path = /var/lib/grafana/dashboards
[security]
admin_user = admin
admin_password = $PASSWORD
[server]
root_url = https://monitoring-${zone}.${domain}
[users]
allow_sign_up = false
auto_assign_org = true
auto_assign_org_role = Editor
" | sudo tee /etc/grafana/grafana.ini > /dev/null
sudo chown grafana:grafana /etc/grafana/grafana.ini

sudo /bin/systemctl restart grafana-server

# Setup prometheus config
# /etc/prometheus/prometheus.yml
echo "
global:
  evaluation_interval: '1m'
  scrape_interval: '30s'
rule_files:
  - /etc/prometheus/api.rules
scrape_configs:
  - job_name: 'prometheus'
    static_configs:
    - targets:
        - 'localhost:9090'
  - job_name: 'pushgateway'
    honor_labels: true
    static_configs:
    - targets:
        - 'localhost:9091'
  - job_name: 'alertmanager'
    static_configs:
    - targets:
        - 'localhost:9093'
  - job_name: 'node-exporter'
    ec2_sd_configs:
      - region: '${region}'
        access_key: ${aws_id}
        secret_key: ${aws_secret}
        port: 9100
  - job_name: 'gateway-http'
    ec2_sd_configs:
      - region: '${region}'
        access_key: ${aws_id}
        secret_key: ${aws_secret}
        port: 9000
  - job_name: 'sims'
    ec2_sd_configs:
      - region: '${region}'
        access_key: ${aws_id}
        secret_key: ${aws_secret}
        port: 9001
  - job_name: 'console'
    ec2_sd_configs:
      - region: '${region}'
        access_key: ${aws_id}
        secret_key: ${aws_secret}
        port: 9002
" | sudo tee /etc/prometheus/prometheus.yml > /dev/null

# /etc/prometheus/api.rules
echo '
job:handler_http_status:sum = sum(rate(handler_request_count [5m])) by (status)
job:handler_http_route:sum = sum(rate(handler_request_count [5m])) by (route)
job:handler_http_latency:apdex = ((sum(rate(handler_request_latency_seconds_bucket{le="0.05"}[5m])) + sum(rate(handler_request_latency_seconds_bucket{le="0.25"}[5m]))) / 2) / sum(rate(handler_request_latency_seconds_count[5m]))
job:handler_http_latency:50 = histogram_quantile(0.5, sum(rate(handler_request_latency_seconds_bucket [5m])) by (le))
job:handler_http_latency:95 = histogram_quantile(0.95, sum(rate(handler_request_latency_seconds_bucket [5m])) by (le))
job:handler_http_latency:99 = histogram_quantile(0.99, sum(rate(handler_request_latency_seconds_bucket [5m])) by (le))
job:service_latency:apdex = ((sum(rate(service_op_latency_seconds_bucket{le="0.005"}[5m])) + sum(rate(service_op_latency_seconds_bucket{le="0.025"}[5m]))) / 2) / sum(rate(service_op_latency_seconds_count[5m]))
job:service_latency:50 = histogram_quantile(0.5, sum(rate(service_op_latency_seconds_bucket [5m])) by (le))
job:service_latency:95 = histogram_quantile(0.95, sum(rate(service_op_latency_seconds_bucket [5m])) by (le))
job:service_latency:99 = histogram_quantile(0.99, sum(rate(service_op_latency_seconds_bucket [5m])) by (le))
job:service_err:count = sum(rate(service_err_count [5m])) by (method, service)
job:service_op:count = sum(rate(service_op_count [5m])) by (method, service)
job:source_latency:apdex = ((sum(rate(source_op_latency_seconds_bucket{le="0.005"}[5m])) + sum(rate(source_op_latency_seconds_bucket{le="0.025"}[5m]))) / 2) / sum(rate(source_op_latency_seconds_count[5m]))
job:source_latency:50 = histogram_quantile(0.5, sum(rate(source_op_latency_seconds_bucket [5m])) by (le))
job:source_latency:95 = histogram_quantile(0.95, sum(rate(source_op_latency_seconds_bucket [5m])) by (le))
job:source_latency:99 = histogram_quantile(0.99, sum(rate(source_op_latency_seconds_bucket [5m])) by (le))
job:source_err:count = sum(rate(source_err_count [5m])) by (method, source)
job:source_op:count = sum(rate(source_op_count [5m])) by (method, source)
job:source_queue_latency:50 = histogram_quantile(0.5, sum(rate(source_queue_latency_seconds_bucket [5m])) by (le))
job:source_queue_latency:95 = histogram_quantile(0.95, sum(rate(source_queue_latency_seconds_bucket [5m])) by (le))
job:source_queue_latency:99 = histogram_quantile(0.99, sum(rate(source_queue_latency_seconds_bucket [5m])) by (le))
job:platform_process_res:sum = sum(process_resident_memory_bytes) by (instance, job)
job:platform_process_cpu:max = max(rate(process_cpu_seconds_total [5m])) by (instance, job)
' | sudo tee /etc/prometheus/api.rules > /dev/null

sudo service prometheus restart