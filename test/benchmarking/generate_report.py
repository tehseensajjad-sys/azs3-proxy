#!/usr/bin/env python3
import json
import sys
import os
import glob
import re
import subprocess
import dateutil.parser

def format_bits(size):
    power = 2**10
    n = size * 8
    power_labels = {0 : '', 1: 'Ki', 2: 'Mi', 3: 'Gi', 4: 'Ti'}
    loop = 0
    while n > power:
        n /= power
        loop += 1
    return f"{n:.2f} {power_labels[loop]}b"

def parse_metrics(csv_path):
    # Dynamic parsing based on header
    if not os.path.exists(csv_path):
        return {}
    
    cpu_vals = []
    mem_vals = []
    net_stats = {} # iface -> {'rx':[], 'tx':[]}
    
    with open(csv_path, 'r') as f:
        lines = f.readlines()
        if len(lines) < 2: return {}
        
        # Parse Header
        headers = lines[0].strip().split(',')
        col_map = {}
        
        for idx, h in enumerate(headers):
            if h.startswith("net_rx_") and h.endswith("_bps"):
                iface = h[7:-4]
                col_map[idx] = ('rx', iface)
                if iface not in net_stats: net_stats[iface] = {'rx':[], 'tx':[]}
            elif h.startswith("net_tx_") and h.endswith("_bps"):
                iface = h[7:-4]
                col_map[idx] = ('tx', iface)
                if iface not in net_stats: net_stats[iface] = {'rx':[], 'tx':[]}
        
        for line in lines[1:]:
            parts = line.strip().split(',')
            if len(parts) < 3: continue
            
            try:
                # cpu, mem
                cpu_vals.append(float(parts[1]))
                mem_vals.append(float(parts[2]) / 1024.0) 
                
                for idx, val in enumerate(parts):
                    if idx in col_map:
                        t, iface = col_map[idx]
                        # Convert Bytes/s to Bits/s
                        net_stats[iface][t].append(float(val) * 8)
            except:
                continue
                
    if not cpu_vals: return {}
    
    def avg(lst): return sum(lst)/len(lst) if lst else 0
    
    res = {
        "cpu_avg": avg(cpu_vals),
        "cpu_max": max(cpu_vals) if cpu_vals else 0,
        "mem_avg": avg(mem_vals),
        "mem_max": max(mem_vals) if mem_vals else 0,
        "network": {}
    }
    
    for iface in net_stats:
        res["network"][iface] = {
            "rx_avg": avg(net_stats[iface]['rx']),
            "tx_avg": avg(net_stats[iface]['tx'])
        }
    
    return res

def main():
    if len(sys.argv) < 2:

        print("Usage: python3 generate_report.py <path_to_zst_files>")
        sys.exit(1)

    workdir = sys.argv[1]
    # Look for .csv.zst or .csv.zst.json.zst files
    files = glob.glob(os.path.join(workdir, "proxy-*.csv.zst"))
    if not files:
        files = glob.glob(os.path.join(workdir, "proxy-*.csv.zst.json.zst"))
    
    if not files:
        print(f"No benchmark files found in {workdir}")
        sys.exit(0)
    
    output_lines = []
    def log(s):
        print(s)
        output_lines.append(s)

    # We need the warp binary location
    warp_bin = "./warp_runs/bin/warp"
    if not os.path.exists(warp_bin):
         # Try finding it relative to the script
         candidate = os.path.join(os.path.dirname(__file__), "../../warp_runs/bin/warp")
         if os.path.exists(candidate):
             warp_bin = candidate
         else:
             print("Warning: Could not find warp binary. If processing raw .csv.zst files, this will fail.")

    log("# Benchmark Summary Report\n")
    # log("| Workload | Concurrency | Throughput (Avg) | Objects/sec (Avg) | Total Requests | Duration |")
    # log("| :--- | :--- | :--- | :--- | :--- | :--- |")

    results = []

    # Sort files naturally
    files.sort()

    for f in files:
        basename = os.path.basename(f)
        # expected format: proxy-{workload}-c{concurrency}.csv.zst
        # or proxy-{workload}-c{concurrency}.csv.zst.json.zst
        
        is_precomputed = f.endswith(".json.zst")
        core_name = basename.replace("proxy-", "").replace(".csv.zst.json.zst", "").replace(".csv.zst", "")

        # Try parse concurrency
        match = re.search(r"(.*)-c(\d+)$", core_name)
        if match:
            workload_name = match.group(1)
            concurrency = match.group(2)
        else:
            workload_name = core_name
            concurrency = "N/A"
        
        name_mapping = {
            "get-1MiB": "Download 1MiB",
            "get-10MiB": "Download 10MiB",
            "get-100MiB": "Download 100MiB",
            "get-2GiB": "Download 2GiB",
            "put-1MiB": "Upload 1MiB",
            "put-10MiB": "Upload 10MiB",
            "put-100MiB": "Upload 100MiB",
            "put-2GiB": "Upload 2GiB",
            "mixed-1MiB": "Mixed Ops 1MiB",
            "small-put-128KiB": "Small Objects 128KiB"
        }
        
        display_name = name_mapping.get(workload_name, workload_name)
        
        if is_precomputed:
             cmd = ["zstd", "-dc", f]
        else:
             cmd = [warp_bin, "analyze", "--json", f]

        try:
            p = subprocess.run(cmd, capture_output=True, text=True, check=True)
            data = json.loads(p.stdout)
            
            total = data.get("total", {})
            if not total:
                # Fallback for empty/failed runs
                log(f"| {workload_name} | {concurrency} | No Data | - | - | - |")
                continue

            reqs = total.get("total_requests", 0)
            
            start = dateutil.parser.isoparse(total["start_time"])
            end = dateutil.parser.isoparse(total["end_time"])
            duration = end - start
            duration_str = str(duration).split('.')[0] # Remove microseconds
            
            total_bytes = total.get("total_bytes", 0)
            duration_sec = duration.total_seconds()
            
            avg_bps = total_bytes / duration_sec if duration_sec > 0 else 0
            avg_ops = reqs / duration_sec if duration_sec > 0 else 0
            
            tput_str = format_bits(avg_bps) + "/s"
            ops_str = f"{avg_ops:.2f} obj/s"
            
            # Look for metrics file
            # e.g. proxy-get-100MiB-c16.csv.zst.metrics.csv
            # The benchmark file may be .csv.zst or .csv.zst.json.zst
            # but metrics file is always based on .csv.zst name
            base_for_metrics = f.replace(".json.zst", "")
            metrics_file = base_for_metrics + ".metrics.csv"
            metrics = parse_metrics(metrics_file)
            
            results.append({
                "name": display_name,
                "concurrency": concurrency,
                "tput": tput_str,
                "ops": ops_str,
                "reqs": reqs,
                "duration": duration_str,
                "metrics": metrics
            })
            
        except Exception as e:
            print(f"Error processing {basename}: {e}")

    # Group results by concurrency
    results_by_conc = {}
    for res in results:
        c = res['concurrency']
        if c not in results_by_conc:
             results_by_conc[c] = []
        results_by_conc[c].append(res)
        
    # Sort concurrencies numerically
    def try_int(x):
        try: return int(x)
        except: return 999999
        
    sorted_concs = sorted(results_by_conc.keys(), key=try_int)
    
    for c in sorted_concs:
        log(f"### Concurrency: {c}")
        log("| Workload | Throughput | Objects/sec | Proxy CPU (Avg) | Proxy Mem (Avg) | Net In (eth0) | Net Out (eth0) | Net In (lo) | Net Out (lo) |")
        log("| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |")
        
        # Sort key helper
        def sort_key(row):
            name = row['name']
            # Order: Download, Upload, Mixed, Others
            type_order = 99
            if name.startswith("Download"): type_order = 1
            elif name.startswith("Upload"): type_order = 2
            elif name.startswith("Mixed"): type_order = 3
            elif name.startswith("Small"): type_order = 4
            
            # Extract size for sorting
            size_val = 0
            match = re.search(r"(\d+)(KiB|MiB|GiB)", name)
            if match:
                val = int(match.group(1))
                unit = match.group(2)
                if unit == "KiB": size_val = val * 1024
                elif unit == "MiB": size_val = val * 1024**2
                elif unit == "GiB": size_val = val * 1024**3
                
            return (type_order, size_val, name)

        # Sort by workload name
        rows = sorted(results_by_conc[c], key=sort_key)
        
        for row in rows:
            m = row.get('metrics', {})
            cpu_s = f"{m.get('cpu_avg', 0):.1f}%" if m else "-"
            mem_s = f"{m.get('mem_avg', 0):.0f} MiB" if m else "-"
            
            # format bits helpers
            def quick_fmt(v):
                if v == 0: return "-"
                return format_bits(v / 8) + "/s" 
            
            net = m.get('network', {})
            
            eth0 = net.get('eth0', {})
            eth0_rx = quick_fmt(eth0.get('rx_avg', 0))
            eth0_tx = quick_fmt(eth0.get('tx_avg', 0))
            
            lo = net.get('lo', {})
            lo_rx = quick_fmt(lo.get('rx_avg', 0))
            lo_tx = quick_fmt(lo.get('tx_avg', 0))
            
            log(f"| **{row['name']}** | **{row['tput']}** | **{row['ops']}** | {cpu_s} | {mem_s} | {eth0_rx} | {eth0_tx} | {lo_rx} | {lo_tx} |")
        
        log("\n")

    with open("BENCHMARK_REPORT.md", "w") as report_file:
         report_file.write("\n".join(output_lines))
    print("Report saved to BENCHMARK_REPORT.md")

if __name__ == "__main__":
    main()
