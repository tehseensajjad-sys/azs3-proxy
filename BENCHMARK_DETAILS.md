# S3-Azure-Proxy Performance Benchmark

## 1. Product Overview

**S3-Azure-Proxy** is a high-performance, lightweight gateway that allows S3-compatible applications to seamlessly interact with Azure Blob Storage. 

### The Problem
Many legacy applications, data processing frameworks (like Spark/Hadoop), and modern tools are built to speak the AWS S3 protocol. Migrating these workloads to Azure often requires significant code refactoring to use Azure SDKs or Blob Storage APIs.

### The Solution
This proxy sits between your S3 client application and Azure Blob Storage. It translates S3 API calls (GET, PUT, LIST, etc.) into Azure Blob Storage REST API calls on the fly. It is designed for high throughput and low latency, utilizing streaming transfers and concurrent connections to maximize link utilization.

## 2. Benchmark Environment

### Infrastructure Details

#### Virtual Machine (Azure)
| Parameter | Value | Description |
| :--- | :--- | :--- |
| **Instance Type** | `Standard_D16s_v5` | Compute optimized instance |
| **vCPUs** | 16 | Intel Ice Lake scalability |
| **Memory** | 64 GiB | Sufficient for large buffer pools |
| **Network Info** | 12.5 Gbps | Measured bandwidth limit to Azure Storage |
| **Region** | South India | Located in same region as storage |
| **OS** | Ubuntu 22.04 LTS | Standard deployment target |

#### Storage Account
| Parameter | Value | Description |
| :--- | :--- | :--- |
| **Type** | Standard General Purpose v2 | Standard Blob Storage |
| **Redundancy** | LRS | Locally Redundant Storage |
| **Region** | South India | Co-located with VM |
| **Auth** | SAS | Low overhead authentication |

#### Software Stack
| Component | Version/Details | Notes |
| :--- | :--- | :--- |
| **S3 Proxy** | `main` branch | Built with Go 1.25.6 |
| **Benchmark Tool** | MinIO WARP v1.4.0 | Industry standard S3 benchmarker |

### Networking Setup
The benchmark controls for network interaction by running the Client (WARP) and the Proxy on the same VM.
*   **Client \ Proxy**: Communicates over `localhost` (Loopback `lo`), eliminating physical network latency between app and proxy.
*   **Proxy \ Azure**: Communicates over `eth0`, representing the real-world WAN/Datacenter link to Azure Storage.

This setup allows us to measure:
1.  **Proxy Overhead**: Difference between `lo` throughput and `eth0` throughput.
2.  **Maximum Potential**: How well the proxy saturates the 12.5 Gbps link given the VM limits.

### Architecture Diagram

```text
       Azure VM (Standard_D16s_v5)
+------------------------------------------+
|                                          |
|   +-----------+          +-----------+   |
|   | S3 Client |   lo     | S3-Azure  |   |
|   |  (WARP)   |<-------->|   Proxy   |   |
|   +-----------+ HTTP/S3  +-----+-----+   |
|                                |         |
+--------------------------------|---------+
                                 | HTTPS/REST
                                 | (eth0)
                                 v
                  +--------------------------+
                  |    Azure Blob Storage    |
                  +--------------------------+
                  Azure Region (South India)
```

## 3. Workloads Executed

| Test Name | Operation | Object Size | Description |
| :--- | :--- | :--- | :--- |
| **Download 1MiB** | GET | 1 MiB | Small object download throughput |
| **Download 10MiB** | GET | 10 MiB | Medium object download throughput |
| **Download 100MiB** | GET | 100 MiB | Large object download throughput |
| **Download 2GiB** | GET | 2 GiB | Very large object download throughput |
| **Upload 1MiB** | PUT | 1 MiB | Small object upload throughput |
| **Upload 10MiB** | PUT | 10 MiB | Medium object upload throughput |
| **Upload 100MiB** | PUT | 100 MiB | Large object upload throughput |
| **Upload 2GiB** | PUT | 2 GiB | Very large object upload throughput |
| **Mixed Ops 1MiB** | Mixed | 1 MiB | 45% GET, 45% PUT, 10% DELETE |
| **Small Objects** | PUT | 128 KiB | High IOPS test for small files |

## 4. Key Metrics Definitions

To interpret the results correctly, we track the following metrics:

*   **Throughput (Gib/s)**: The aggregate amount of data transferred per second. High throughput indicates efficient link utilization.
*   **Objects/s (IOPS)**: The number of successful S3 operations completed per second. Critical for small-file workloads.
*   **Time To First Byte (TTFB)**: The latency from when a request is sent until the first byte of data is received. Indicates proxy processing overhead.
*   **Net In / Net Out**:
    *   `eth0`: Traffic on the external network interface. Represents actual data flow to Azure.
    *   `lo`: Traffic on the loopback interface. Represents application-perceived throughput.
*   **Proxy CPU & Memory**: Resource consumption of the proxy process during the test.

## 5. Results Summary

### Consolidated Report
The following table summarizes the performance across different concurrencies (8, 16, 64).

*(Results will be automatically generated and appended here after the run completes)*
