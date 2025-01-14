package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/ini.v1"
)

const cfgFile = "/etc/libvirt/hooks/config.ini"

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: ./script <VM_NAME> <start|stopped|reconnect>")
		return
	}

	vmName := os.Args[1]
	action := os.Args[2]

	hostIP, config, err := parseINIConfig(cfgFile)
	if err != nil {
		fmt.Printf("Error reading config.ini file: %v\n", err)
		return
	}

	entries, ok := config[vmName]
	if !ok {
		fmt.Printf("VM '%s' not found in config.ini file\n", vmName)
		return
	}

	for _, entry := range entries {
		guestIP, guestPort, hostPort, allow, err := parseRule(entry)
		if err != nil {
			fmt.Printf("Invalid rule '%s': %v\n", entry, err)
			continue
		}

		switch action {
		case "stopped":
			removeIptablesRule(hostIP, guestIP, guestPort, hostPort, allow)
		case "start":
			addIptablesRule(hostIP, guestIP, guestPort, hostPort, allow)
		case "reconnect":
			removeIptablesRule(hostIP, guestIP, guestPort, hostPort, allow)
			addIptablesRule(hostIP, guestIP, guestPort, hostPort, allow)
		default:
			fmt.Printf("Invalid action: %s\n", action)
		}
	}
}

func parseINIConfig(path string) (string, map[string][]string, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return "", nil, err
	}

	hostIP := cfg.Section("DEFAULT").Key("host_ip").String()
	if hostIP == "" {
		return "", nil, fmt.Errorf("host_ip not found in DEFAULT section")
	}

	config := make(map[string][]string)
	for _, section := range cfg.Sections() {
		if section.Name() == "DEFAULT" {
			continue
		}
		for _, key := range section.Keys() {
			config[section.Name()] = append(config[section.Name()], key.Value())
		}
	}

	return hostIP, config, nil
}

func parseRule(rule string) (guestIP, guestPort, hostPort, allow string, err error) {
	parts := strings.Split(rule, ",")
	rulePart := parts[0]

	guestIP, guestPort, hostPort, err = parseCoreRule(rulePart)
	if err != nil {
		return "", "", "", "", err
	}

	for _, option := range parts[1:] {
		if strings.HasPrefix(option, "allow:") {
			allow = strings.TrimPrefix(option, "allow:")
		}
	}

	return guestIP, guestPort, hostPort, allow, nil
}

func parseCoreRule(rule string) (guestIP, guestPort, hostPort string, err error) {
	parts := strings.Split(rule, "->")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid format, expected 'IP:PORT->PORT'")
	}

	guestPart := parts[0]
	hostPort = parts[1]

	guestParts := strings.Split(guestPart, ":")
	if len(guestParts) != 2 {
		return "", "", "", fmt.Errorf("invalid guest format, expected 'IP:PORT'")
	}

	guestIP = guestParts[0]
	guestPort = guestParts[1]

	return guestIP, guestPort, hostPort, nil
}

func addIptablesRule(hostIP, guestIP, guestPort, hostPort, allow string) {
	commands := [][]string{
		{"-I", "FORWARD", "-o", "virbr0", "-p", "tcp", "-d", guestIP, "--dport", guestPort, "-j", "ACCEPT"},
		{"-t", "nat", "-I", "PREROUTING", "-p", "tcp", "-d", hostIP, "--dport", hostPort, "-j", "DNAT", "--to", fmt.Sprintf("%s:%s", guestIP, guestPort)},
	}

	if allow != "" {
		commands[0] = append(commands[0], "-s", allow)
	}

	for _, cmd := range commands {
		if err := runCommand("iptables", cmd...); err != nil {
			fmt.Printf("Error adding iptables rule: %v\n", err)
		}
	}
}

func removeIptablesRule(hostIP, guestIP, guestPort, hostPort, allow string) {
	commands := [][]string{
		{"-D", "FORWARD", "-o", "virbr0", "-p", "tcp", "-d", guestIP, "--dport", guestPort, "-j", "ACCEPT"},
		{"-t", "nat", "-D", "PREROUTING", "-p", "tcp", "-d", hostIP, "--dport", hostPort, "-j", "DNAT", "--to", fmt.Sprintf("%s:%s", guestIP, guestPort)},
	}

	if allow != "" {
		commands[0] = append(commands[0], "-s", allow)
	}

	for _, cmd := range commands {
		if err := runCommand("iptables", cmd...); err != nil {
			fmt.Printf("Error removing iptables rule: %v\n", err)
		}
	}
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
