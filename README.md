# qemu-hook-go
Qemu hook for NAT on Golang

is used in conjunction with libvirt to automatically manage iptables rules when the state 
of QEMU virtual machines changes. The utility adds or removes rules for port forwarding 
between the host and guest machines 
[more about this](https://wiki.libvirt.org/Networking.html#forwarding-incoming-connections).

### How to Use

Create the `config.yaml` file: Place the configuration file in the `/etc/libvirt/hooks/` directory. 
Example:
```yaml
default:
  host_ip: 192.168.1.1

vms:
  - name: VM1
    rules:
      - guest_ip: 192.168.122.224
        guest_ports: "443"
        host_ports: "443"
        allow: 192.168.0.0/24
        protocol: tcp
      - guest_ip: 192.168.122.224
        guest_ports: "80:100"
        host_ports: "8080:8090"
        protocol: udp

  - name: VM2
    rules:
      - guest_ip: 192.168.122.225
        guest_ports: "22"
        host_ports: "2222"
        allow: 192.168.0.100
        protocol: tcp
```

Install the QEMU hook: Place the compiled script in the libvirt hooks directory:
```shell
/etc/libvirt/hooks/qemu
```
Make the file executable:
```shell
chmod +x /etc/libvirt/hooks/qemu
```

When a virtual machine starts, stops, or restarts, libvirt will automatically invoke the hook.

### Note
A Linux x64 compiled file is available in the Releases section.
