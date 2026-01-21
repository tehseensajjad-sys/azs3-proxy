#!/usr/bin/env python3
import sys
import time
import os
import signal

# Global flag to run loop
running = True

def handle_sigterm(*args):
    global running
    running = False

signal.signal(signal.SIGTERM, handle_sigterm)

def get_cpu_times(pid):
    try:
        with open(f"/proc/{pid}/stat", 'r') as f:
            fields = f.read().split()
            # utime(13) + stime(14)
            return int(fields[13]) + int(fields[14])
    except:
        return 0

def get_system_cpu_times():
    try:
        with open("/proc/stat", 'r') as f:
            line = f.readline() 
            fields = line.split()
            # sum all columns after 'cpu'
            return sum(int(x) for x in fields[1:])
    except:
        return 0

def get_memory(pid):
    # Read VmRSS from /proc/pid/status
    try:
        with open(f"/proc/{pid}/status", 'r') as f:
            for line in f:
                if line.startswith("VmRSS:"):
                    return int(line.split()[1]) # kB
    except:
        return 0
    return 0

def get_network(interface_arg="eth0"):
    # interface_arg can be "eth0" or "eth0,lo" etc.
    # Returns a dictionary { interface_name: (rx_bytes, tx_bytes) }
    
    iface_names = []
    if interface_arg == "all":
        try:
            all_ifaces = os.listdir('/sys/class/net')
            for iface in all_ifaces:
                if iface.startswith('lo') or iface.startswith('docker') or iface.startswith('veth'):
                    continue
                iface_names.append(iface)
        except:
            iface_names = ["eth0"]
    else:
        iface_names = interface_arg.split(',')
        
    stats = {}
    for iface in iface_names:
        rx = 0
        tx = 0
        try:
            with open(f"/sys/class/net/{iface}/statistics/rx_bytes", 'r') as f:
                rx = int(f.read())
            with open(f"/sys/class/net/{iface}/statistics/tx_bytes", 'r') as f:
                tx = int(f.read())
        except:
            pass
        stats[iface] = (rx, tx)
            
    return stats


def main():
    if len(sys.argv) < 3:
        print("Usage: monitor_system.py <pid> <output_csv> [interface]")
        sys.exit(1)
        
    pid = sys.argv[1]
    output_file = sys.argv[2]
    interface_arg = sys.argv[3] if len(sys.argv) > 3 else "eth0"
    
    # Check what interfaces we found
    initial_stats = get_network(interface_arg)
    monitored_interfaces = sorted(initial_stats.keys())
    
    # Write Header
    header = "timestamp,cpu_percent,mem_rss_kb"
    for iface in monitored_interfaces:
        header += f",net_rx_{iface}_bps,net_tx_{iface}_bps"
    
    with open(output_file, 'w') as f:
        f.write(header + "\n")
    
    # Initial readings
    prev_proc_cpu = get_cpu_times(pid)
    prev_sys_cpu = get_system_cpu_times()
    prev_net_stats = initial_stats
    prev_time = time.time()
    
    num_cpus = os.cpu_count() or 1
    
    # Sleep a bit to get a delta
    time.sleep(1)
    
    while running:
        curr_time = time.time()
        time_delta = curr_time - prev_time
        if time_delta <= 0: time_delta = 1.0 # fallback
        
        curr_proc_cpu = get_cpu_times(pid)
        curr_sys_cpu = get_system_cpu_times()
        curr_net_stats = get_network(interface_arg)
        mem = get_memory(pid)
        
        # CPU calculation
        proc_delta = curr_proc_cpu - prev_proc_cpu
        sys_delta = curr_sys_cpu - prev_sys_cpu
        
        cpu_usage = 0.0
        if sys_delta > 0:
            cpu_usage = (proc_delta / sys_delta) * num_cpus * 100.0
            
        # Network calculation
        net_metrics_str = ""
        for iface in monitored_interfaces:
            p_rx, p_tx = prev_net_stats.get(iface, (0,0))
            c_rx, c_tx = curr_net_stats.get(iface, (0,0))
            
            rx_bps = (c_rx - p_rx) / time_delta
            tx_bps = (c_tx - p_tx) / time_delta
            
            net_metrics_str += f",{rx_bps:.2f},{tx_bps:.2f}"
            
        with open(output_file, 'a') as f:
            f.write(f"{curr_time:.2f},{cpu_usage:.2f},{mem}{net_metrics_str}\n")
            
        sys.stdout.flush()
        
        prev_proc_cpu = curr_proc_cpu
        prev_sys_cpu = curr_sys_cpu
        prev_net_stats = curr_net_stats
        prev_time = curr_time
        
        time.sleep(1)

if __name__ == "__main__":
    main()
