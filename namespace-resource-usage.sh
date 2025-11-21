#!/bin/bash

# Script to find Kubernetes namespaces with highest CPU and memory usage
# This script only performs read-only operations

set -euo pipefail

echo "=================================================="
echo "Kubernetes Namespace Resource Usage Report"
echo "=================================================="
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl not found. Please install kubectl first."
    exit 1
fi

# Check if metrics-server is available
if ! kubectl top nodes &> /dev/null; then
    echo "Error: Metrics server is not available or not responding."
    echo "Please ensure metrics-server is installed and running in your cluster."
    exit 1
fi

echo "Collecting pod metrics across all namespaces..."
echo ""

# Get all pod metrics and aggregate by namespace
# Using kubectl top pods --all-namespaces gives us current CPU and memory usage
pod_metrics=$(kubectl top pods --all-namespaces --no-headers 2>/dev/null)

if [ -z "$pod_metrics" ]; then
    echo "No pod metrics available."
    exit 0
fi

# Create temporary files for processing
cpu_file=$(mktemp)
mem_file=$(mktemp)
combined_file=$(mktemp)

# Clean up temp files on exit
trap "rm -f $cpu_file $mem_file $combined_file" EXIT

# Process metrics and aggregate by namespace
echo "$pod_metrics" | awk '{
    namespace = $1
    cpu = $3
    memory = $4

    # Convert CPU (millicores to cores)
    cpu_value = cpu
    if (cpu ~ /m$/) {
        gsub(/m/, "", cpu_value)
        cpu_cores = cpu_value / 1000
    } else {
        cpu_cores = cpu_value
    }

    # Convert memory to MB
    mem_value = memory
    mem_mb = 0
    if (mem_value ~ /Ki$/) {
        gsub(/Ki/, "", mem_value)
        mem_mb = mem_value / 1024
    } else if (mem_value ~ /Mi$/) {
        gsub(/Mi/, "", mem_value)
        mem_mb = mem_value
    } else if (mem_value ~ /Gi$/) {
        gsub(/Gi/, "", mem_value)
        mem_mb = mem_value * 1024
    } else {
        # Assume bytes
        mem_mb = mem_value / 1024 / 1024
    }

    cpu_total[namespace] += cpu_cores
    mem_total[namespace] += mem_mb
    pod_count[namespace]++
}
END {
    for (ns in cpu_total) {
        printf "%s %.3f %.2f %d\n", ns, cpu_total[ns], mem_total[ns], pod_count[ns]
    }
}' > "$combined_file"

# Sort by CPU usage (descending)
echo "TOP 10 NAMESPACES BY CPU USAGE:"
echo "================================"
printf "%-40s %15s %15s %10s\n" "NAMESPACE" "CPU (cores)" "MEMORY (MB)" "PODS"
echo "--------------------------------------------------------------------------------"
sort -t' ' -k2 -rn "$combined_file" | head -10 | while read ns cpu mem pods; do
    printf "%-40s %15.3f %15.2f %10d\n" "$ns" "$cpu" "$mem" "$pods"
done

echo ""
echo ""

# Sort by Memory usage (descending)
echo "TOP 10 NAMESPACES BY MEMORY USAGE:"
echo "==================================="
printf "%-40s %15s %15s %10s\n" "NAMESPACE" "CPU (cores)" "MEMORY (MB)" "PODS"
echo "--------------------------------------------------------------------------------"
sort -t' ' -k3 -rn "$combined_file" | head -10 | while read ns cpu mem pods; do
    printf "%-40s %15.3f %15.2f %10d\n" "$ns" "$cpu" "$mem" "$pods"
done

echo ""
echo ""

# Calculate totals
total_cpu=$(awk '{sum += $2} END {printf "%.3f", sum}' "$combined_file")
total_mem=$(awk '{sum += $3} END {printf "%.2f", sum}' "$combined_file")
total_namespaces=$(wc -l < "$combined_file")

echo "SUMMARY:"
echo "========"
echo "Total namespaces with pods: $total_namespaces"
echo "Total CPU usage: ${total_cpu} cores"
echo "Total Memory usage: ${total_mem} MB ($(awk "BEGIN {printf \"%.2f\", $total_mem/1024}") GB)"
echo ""
