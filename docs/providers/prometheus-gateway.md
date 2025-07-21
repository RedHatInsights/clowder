# Prometheus Gateway Provider

The Prometheus Gateway provider in Clowder enables the deployment and management of Prometheus Pushgateway instances alongside your Prometheus monitoring setup. This is particularly useful for collecting metrics from short-lived jobs, batch processes, and ephemeral applications that don't run long enough to be scraped by Prometheus directly.

## Overview

Prometheus Pushgateway acts as an intermediary service where applications can push their metrics. Prometheus then scrapes these metrics from the Pushgateway. This is especially valuable for:

- Batch jobs and cron tasks
- Short-lived containers or functions
- Services with high turnover rates
- Applications in environments where pull-based monitoring is challenging

## Configuration

To enable Prometheus Gateway in your ClowdEnvironment, you need to set the `prometheusGateway.deploy` field to `true` in the metrics provider configuration:

```yaml
apiVersion: cloud.redhat.com/v1alpha1
kind: ClowdEnvironment
metadata:
  name: my-environment
spec:
  providers:
    metrics:
      mode: operator
      prometheus:
        deploy: true
      prometheusGateway:
        deploy: true
```

## What Gets Created

When you enable the Prometheus Gateway provider, Clowder will create the following resources in your target namespace:

1. **Deployment**: A Prometheus Pushgateway deployment running the official `prom/pushgateway` image
2. **Service**: A Kubernetes service exposing the Pushgateway on port 9091
3. **ServiceMonitor**: A Prometheus Operator ServiceMonitor resource to enable automatic discovery by Prometheus

## Resource Configuration

The Prometheus Gateway deployment is created with the following default configuration:

- **Image**: `prom/pushgateway:latest`
- **Port**: 9091 (standard Pushgateway port)
- **Resources**:
  - CPU Requests: 50m
  - CPU Limits: 100m
  - Memory Requests: 128Mi
  - Memory Limits: 256Mi
- **Replicas**: 1

## Using the Prometheus Gateway

Once deployed, applications can push metrics to the Pushgateway using the standard Prometheus client libraries or simple HTTP requests.

### Example: Pushing Metrics with curl

```bash
# Push a simple metric
echo "job_duration_seconds 45.2" | curl --data-binary @- \
  http://<environment-name>-prometheus-gateway.<namespace>:9091/metrics/job/my-batch-job

# Push multiple metrics with metadata
cat <<EOF | curl --data-binary @- \
  http://<environment-name>-prometheus-gateway.<namespace>:9091/metrics/job/my-batch-job/instance/worker-1
# TYPE job_duration_seconds gauge
# HELP job_duration_seconds Duration of the batch job in seconds
job_duration_seconds 45.2
# TYPE processed_records_total counter
# HELP processed_records_total Total number of records processed
processed_records_total 1234
EOF
```

### Example: Using Python Client

```python
from prometheus_client import CollectorRegistry, Gauge, push_to_gateway

registry = CollectorRegistry()
duration_gauge = Gauge('job_duration_seconds', 'Job duration', registry=registry)
duration_gauge.set(45.2)

push_to_gateway('my-environment-prometheus-gateway.my-namespace:9091', 
                job='my-batch-job', registry=registry)
```

## Integration with Prometheus

The Prometheus Gateway automatically integrates with your Prometheus instance through:

1. **ServiceMonitor Creation**: When `createServiceMonitor` is enabled in your Clowder configuration, a ServiceMonitor is automatically created with the label `prometheus: <environment-name>`.

2. **Automatic Discovery**: Your Prometheus instance will automatically discover and scrape the Pushgateway based on the ServiceMonitor selector configuration.

3. **Metric Preservation**: Metrics pushed to the gateway are exposed with their original labels, plus any grouping labels specified in the push URL.

## Best Practices

1. **Use Appropriate Job Names**: Use descriptive job names in your push URLs to easily identify metrics in Prometheus.

2. **Include Instance Labels**: For jobs that run on multiple instances, include instance identifiers in your push URLs.

3. **Clean Up Metrics**: Delete metrics from the Pushgateway when jobs complete to avoid stale data:
   ```bash
   curl -X DELETE http://gateway:9091/metrics/job/my-batch-job/instance/worker-1
   ```

4. **Monitor Push Success**: Set up alerts on `push_time_seconds` and `push_failure_time_seconds` metrics to monitor the health of your batch jobs.

## Limitations

- The Prometheus Gateway does not provide strong consistency guarantees
- Metrics are stored in memory and will be lost if the pod restarts (unless persistence is configured)
- High cardinality metrics can impact performance

## Related Configuration

- **Prometheus Provider**: The Prometheus Gateway works alongside the Prometheus provider. Ensure both are properly configured.
- **ServiceMonitor Creation**: The `createServiceMonitor` feature flag in your Clowder configuration affects whether ServiceMonitors are created.

## Example ClowdEnvironment

See the complete example in `config/samples/prometheus-gateway-example.yaml` for a full ClowdEnvironment configuration with Prometheus Gateway enabled. 