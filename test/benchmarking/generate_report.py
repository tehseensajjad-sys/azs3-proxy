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
            "get-100MiB": "Large Download",
            "get-10MiB": "Download",
            "mixed-1MiB": "Mixed Ops",
            "put-10MiB": "Upload",
            "put-100MiB": "Large Upload",
            "put-2GiB": "Very Large Upload",
            "small-put-128KiB": "Small Objects"
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
            
            results.append({
                "name": display_name,
                "concurrency": concurrency,
                "tput": tput_str,
                "ops": ops_str,
                "reqs": reqs,
                "duration": duration_str
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
        log("| Workload | Throughput (Avg) | Objects/sec (Avg) | Total Requests | Duration |")
        log("| :--- | :--- | :--- | :--- | :--- |")
        
        # Sort by workload name
        rows = sorted(results_by_conc[c], key=lambda x: x['name'])
        
        for row in rows:
            log(f"| **{row['name']}** | **{row['tput']}** | **{row['ops']}** | {row['reqs']} | {row['duration']} |")
        
        log("\n")

    with open("BENCHMARK_REPORT.md", "w") as report_file:
         report_file.write("\n".join(output_lines))
    print("Report saved to BENCHMARK_REPORT.md")

if __name__ == "__main__":
    main()
