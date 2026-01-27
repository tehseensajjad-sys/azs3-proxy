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

## 5. Industry Reference (Baselines)

To provide context for S3-Azure-Proxy's performance, we refer to industry benchmarks for standard S3 performance.

**Source**: [Rabata.io S3 Comparison](https://rabata.io/s3-comparison) (Tested on US-East-1, Debian VM, MinIO WARP v1.0.7, **Concurrency: 8**)

| Provider | Upload Speed (PUT) | Download Speed (GET) | Mixed (1MB) | Small Objects (PUT) |
| :--- | :--- | :--- | :--- | :--- |
| **Amazon S3** | **1,444 MB/s** (~11.3 Gbps) | **1,816 MB/s** (~14.2 Gbps) | **151 MB/s** | **319 obj/s** |
| **Rabata.io** | 1,462 MB/s | 1,107 MB/s | 346 MB/s | 696 obj/s |
| **DigitalOcean**| 1,440 MB/s | 1,728 MB/s | 179 MB/s | 328 obj/s |

*Note: These are baseline numbers for native object storage services. Our benchmarks measure the performance of the **Proxy layer** sitting in front of Azure Blob Storage.*

## 6. Results Summary

### Consolidated Report
The following table summarizes the performance across different concurrencies (8, 16, 64).

# Benchmark Summary Report

### Concurrency: 8
| Workload | Throughput | Objects/sec | Proxy CPU (Avg) | Proxy Mem (Avg) | Network In | Network Out |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Download 1MiB** | **1.34 Gib/s** | **171.10 obj/s** | 131.0% | 157 MiB | 1.17 Gib/s | 96.15 Mib/s |
| **Download 10MiB** | **5.12 Gib/s** | **65.54 obj/s** | 106.4% | 210 MiB | 3.61 Gib/s | 761.59 Mib/s |
| **Download 100MiB** | **24.61 Gib/s** | **31.50 obj/s** | 499.9% | 1274 MiB | 11.50 Gib/s | 4.93 Gib/s |
| **Download 2GiB** | **32.24 Gib/s** | **2.01 obj/s** | 242.7% | 1339 MiB | 1.61 Gib/s | 10.51 Gib/s |
| **Upload 1MiB** | **847.30 Mib/s** | **105.91 obj/s** | 46.7% | 166 MiB | 2.02 Mib/s | 498.74 Mib/s |
| **Upload 10MiB** | **1.71 Gib/s** | **21.86 obj/s** | 32.3% | 313 MiB | 3.16 Mib/s | 1.47 Gib/s |
| **Upload 100MiB** | **6.41 Gib/s** | **8.21 obj/s** | 109.8% | 977 MiB | 11.99 Mib/s | 5.99 Gib/s |
| **Upload 2GiB** | **7.28 Gib/s** | **0.45 obj/s** | 120.0% | 1156 MiB | 16.16 Mib/s | 7.16 Gib/s |
| **Mixed Ops 1MiB** | **1.22 Gib/s** | **260.18 obj/s** | 97.5% | 179 MiB | 751.80 Mib/s | 328.81 Mib/s |
| **small-128KiB** | **713.72 Mib/s** | **713.72 obj/s** | 69.2% | 158 MiB | 2.93 Mib/s | 125.48 Mib/s |


### Concurrency: 16
| Workload | Throughput | Objects/sec | Proxy CPU (Avg) | Proxy Mem (Avg) | Network In | Network Out |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Download 1MiB** | **2.56 Gib/s** | **327.72 obj/s** | 200.2% | 154 MiB | 2.26 Gib/s | 100.51 Mib/s |
| **Download 10MiB** | **8.70 Gib/s** | **111.38 obj/s** | 173.7% | 267 MiB | 6.27 Gib/s | 789.40 Mib/s |
| **Download 100MiB** | **30.57 Gib/s** | **39.14 obj/s** | 611.7% | 2320 MiB | 15.10 Gib/s | 5.23 Gib/s |
| **Download 2GiB** | **31.11 Gib/s** | **1.94 obj/s** | 265.7% | 2591 MiB | 1.57 Gib/s | 10.54 Gib/s |
| **Upload 1MiB** | **2.20 Gib/s** | **282.06 obj/s** | 57.1% | 159 MiB | 3.25 Mib/s | 804.26 Mib/s |
| **Upload 10MiB** | **6.25 Gib/s** | **79.96 obj/s** | 87.1% | 395 MiB | 9.22 Mib/s | 4.30 Gib/s |
| **Upload 100MiB** | **11.05 Gib/s** | **14.15 obj/s** | 210.4% | 1767 MiB | 25.39 Mib/s | 10.04 Gib/s |
| **Upload 2GiB** | **11.00 Gib/s** | **0.69 obj/s** | 224.1% | 2138 MiB | 34.81 Mib/s | 10.79 Gib/s |
| **Mixed Ops 1MiB** | **1.84 Gib/s** | **391.66 obj/s** | 111.6% | 176 MiB | 1.10 Gib/s | 450.19 Mib/s |
| **small-128KiB** | **1.45 Gib/s** | **1488.69 obj/s** | 61.2% | 148 MiB | 3.00 Mib/s | 156.07 Mib/s |


### Concurrency: 64
| Workload | Throughput | Objects/sec | Proxy CPU (Avg) | Proxy Mem (Avg) | Network In | Network Out |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Download 1MiB** | **13.25 Gib/s** | **1696.19 obj/s** | 599.7% | 164 MiB | 12.30 Gib/s | 138.46 Mib/s |
| **Download 10MiB** | **31.41 Gib/s** | **402.08 obj/s** | 802.2% | 361 MiB | 27.22 Gib/s | 1.00 Gib/s |
| **Download 100MiB** | **30.82 Gib/s** | **39.45 obj/s** | 610.0% | 8018 MiB | 14.51 Gib/s | 5.16 Gib/s |
| **Download 2GiB** | **29.79 Gib/s** | **1.86 obj/s** | 304.5% | 10666 MiB | 1.69 Gib/s | 10.63 Gib/s |
| **Upload 1MiB** | **5.77 Gib/s** | **738.31 obj/s** | 42.3% | 153 MiB | 4.01 Mib/s | 992.13 Mib/s |
| **Upload 10MiB** | **11.05 Gib/s** | **141.43 obj/s** | 116.1% | 1017 MiB | 13.09 Mib/s | 5.75 Gib/s |
| **Upload 100MiB** | **10.92 Gib/s** | **13.97 obj/s** | 226.3% | 6653 MiB | 74.24 Mib/s | 9.10 Gib/s |
| **Upload 2GiB** | **10.93 Gib/s** | **0.68 obj/s** | 260.5% | 8197 MiB | 101.66 Mib/s | 10.93 Gib/s |
| **Mixed Ops 1MiB** | **6.90 Gib/s** | **1472.57 obj/s** | 211.7% | 204 MiB | 3.28 Gib/s | 1.13 Gib/s |
| **small-128KiB** | **4.23 Gib/s** | **4327.97 obj/s** | 40.7% | 165 MiB | 10.81 Mib/s | 145.86 Mib/s |


