"""Gaia simple prometheus exporter"""

import os
import time
import requests
import argparse
import prometheus_client as prom

class AppMetrics:
    def __init__ (self, target_host='localhost', target_port=26657, polling_interval_seconds=15):
        self.target_host = target_host
        self.target_port = target_port
        self.polling_interval_seconds = polling_interval_seconds
        
        # Prometheus metrics to collect
        self.latest_block_height = prom.Gauge('latest_block_height', 'Latest block height')
        self.latest_block_time_lag = prom.Gauge('latest_block_time_lag', 'Latest block time lag')
        self.number_of_peers = prom.Gauge('number_of_peers', 'Number of peers')

    def fetch(self):
        # Fetch raw status data from the application
        resp_status = requests.get(url=f"http://{self.target_host}:{self.target_port}/status")
        status_data = resp_status.json()

        resp_netinfo = requests.get(url=f"http://{self.target_host}:{self.target_port}/net_info")
        netinfo_data = resp_netinfo.json()

        # convert "2019-01-01T00:00:00Z" to unix timestamp  
        latest_block_time_unix = time.mktime(time.strptime(status_data['result']['sync_info']['latest_block_time'][:-4], "%Y-%m-%dT%H:%M:%S.%f"))
        current_time_unix = time.time()

        # Update Prometheus metrics with application metrics
        self.latest_block_height.set(status_data['result']['sync_info']['latest_block_height'])
        self.latest_block_time_lag.set(current_time_unix - latest_block_time_unix)
        self.number_of_peers.set(netinfo_data['result']['n_peers'])

    def run_metrics_loop(self):
        while True:
            self.fetch()
            time.sleep(self.polling_interval_seconds)

def main():
    # Parse command line arguments
    parser = argparse.ArgumentParser(description='Gaia simple prometheus exporter')
    parser.add_argument('--host', type=str, default='localhost', help='Hostname of the application')
    parser.add_argument('--port', type=int, default=26657, help='Port of the application')
    parser.add_argument('--polling-interval', type=int, default=15, help='Polling interval in seconds')
    parser.add_argument('--listen-port', type=int, default=9090, help='Port to listen on')
    args = parser.parse_args()

    # Disable unneeded metrics
    prom.REGISTRY.unregister(prom.PROCESS_COLLECTOR)
    prom.REGISTRY.unregister(prom.PLATFORM_COLLECTOR)
    prom.REGISTRY.unregister(prom.GC_COLLECTOR)

    # Start the metrics server
    prom.start_http_server(args.listen_port)

    # Create the metrics object
    metrics = AppMetrics(args.host, args.port, args.polling_interval)

    # Start the metrics loop
    metrics.run_metrics_loop()

if __name__ == '__main__':
    main()

