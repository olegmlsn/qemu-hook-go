# qemu-hook-go
Qemu hook for NAT on Golang

is used in conjunction with libvirt to automatically manage iptables rules when the state 
of QEMU virtual machines changes. The utility adds or removes rules for port forwarding 
between the host and guest machines 
[more about this](https://wiki.libvirt.org/Networking.html#forwarding-incoming-connections).

### How to Use

Create the `config.ini` file: Place the configuration file in the `/etc/libvirt/hooks/` directory. 
Example:
```ini
[DEFAULT]
host_ip = 192.168.1.1

[VM1]
;; GUEST_IP:GUEST_PORT->HOST_PORT|allow:ALLOWED_IP|protocol:tcp or udp
rule1 = 192.168.122.224:443->443
rule2 = 192.168.122.224:69->2269|protocol:udp
rule1 = 192.168.122.224:5060->5060|allow:192.168.0.100|protocol:udp

[VM2]
rule1 = 192.168.122.225:22->2222|allow:192.168.0.0/24
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
