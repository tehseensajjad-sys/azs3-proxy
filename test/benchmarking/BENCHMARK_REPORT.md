# Benchmark Summary Report

| Workload | Concurrency | Throughput (Avg) | Objects/sec (Avg) | Total requests | Duration |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **get-1MiB** | 16 | **325.31 MiB/s** | **325.31 obj/s** | 97630 | 0:05:00 |
| **get-1MiB** | 64 | **1.72 GiB/s** | **1764.50 obj/s** | 530112 | 0:05:00 |
| **get-1MiB** | 8 | **153.70 MiB/s** | **153.70 obj/s** | 46122 | 0:05:00 |
| **mixed-1MiB** | 16 | **242.62 MiB/s** | **404.32 obj/s** | 121446 | 0:05:00 |
| **mixed-1MiB** | 64 | **1.08 GiB/s** | **1845.75 obj/s** | 554184 | 0:05:00 |
| **mixed-1MiB** | 8 | **140.38 MiB/s** | **234.00 obj/s** | 70245 | 0:05:00 |
| **put-1MiB** | 16 | **293.26 MiB/s** | **293.26 obj/s** | 88021 | 0:05:00 |
| **put-1MiB** | 64 | **860.67 MiB/s** | **860.67 obj/s** | 258521 | 0:05:00 |
| **put-1MiB** | 8 | **94.11 MiB/s** | **94.11 obj/s** | 28313 | 0:05:00 |
| **small-put-128KiB** | 16 | **172.39 MiB/s** | **1379.12 obj/s** | 413806 | 0:05:00 |
| **small-put-128KiB** | 64 | **105.53 MiB/s** | **844.24 obj/s** | 253421 | 0:05:00 |
| **small-put-128KiB** | 8 | **94.08 MiB/s** | **752.61 obj/s** | 225793 | 0:05:00 |

---

### Detailed Breakdown

#### get-1MiB (Concurrency: 16)
* **Throughput:** 325.31 MiB/s
* **Requests:** 325.31 obj/s


#### get-1MiB (Concurrency: 64)
* **Throughput:** 1.72 GiB/s
* **Requests:** 1764.50 obj/s


#### get-1MiB (Concurrency: 8)
* **Throughput:** 153.70 MiB/s
* **Requests:** 153.70 obj/s


#### mixed-1MiB (Concurrency: 16)
* **Throughput:** 242.62 MiB/s
* **Requests:** 404.32 obj/s


#### mixed-1MiB (Concurrency: 64)
* **Throughput:** 1.08 GiB/s
* **Requests:** 1845.75 obj/s


#### mixed-1MiB (Concurrency: 8)
* **Throughput:** 140.38 MiB/s
* **Requests:** 234.00 obj/s


#### put-1MiB (Concurrency: 16)
* **Throughput:** 293.26 MiB/s
* **Requests:** 293.26 obj/s


#### put-1MiB (Concurrency: 64)
* **Throughput:** 860.67 MiB/s
* **Requests:** 860.67 obj/s


#### put-1MiB (Concurrency: 8)
* **Throughput:** 94.11 MiB/s
* **Requests:** 94.11 obj/s


#### small-put-128KiB (Concurrency: 16)
* **Throughput:** 172.39 MiB/s
* **Requests:** 1379.12 obj/s


#### small-put-128KiB (Concurrency: 64)
* **Throughput:** 105.53 MiB/s
* **Requests:** 844.24 obj/s


#### small-put-128KiB (Concurrency: 8)
* **Throughput:** 94.08 MiB/s
* **Requests:** 752.61 obj/s
