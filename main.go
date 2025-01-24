package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Default struct {
		HostIP string `yaml:"host_ip"`
	} `yaml:"default"`
	VMs []VM `yaml:"vms"`
}

type VM struct {
	Name  string `yaml:"name"`
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	GuestIP    string `yaml:"guest_ip"`
	GuestPorts string `yaml:"guest_ports"`
	HostPorts  string `yaml:"host_ports"`
	Allow      string `yaml:"allow,omitempty"`
	Protocol   string `yaml:"protocol"`
}

const cfgFile = "/etc/libvirt/hooks/config.yaml"

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: ./script <VM_NAME> <start|stopped|reconnect>")
		return
	}

	vmName := os.Args[1]
	action := os.Args[2]

	config, err := loadYAMLConfig(cfgFile)
	if err != nil {
		fmt.Printf("Error loading YAML config: %v\n", err)
		return
	}

	hostIP := config.Default.HostIP
	var vm *VM
	for _, v := range config.VMs {
		if v.Name == vmName {
			vm = &v
			break
		}
	}

	if vm == nil {
		fmt.Printf("VM '%s' not found in config\n", vmName)
		return
	}

	for _, rule := range vm.Rules {
		switch action {
		case "stopped":
			removeIptablesRule(hostIP, rule)
		case "start":
			addIptablesRule(hostIP, rule)
		case "reconnect":
			removeIptablesRule(hostIP, rule)
			addIptablesRule(hostIP, rule)
		default:
			fmt.Printf("Invalid action: %s\n", action)
		}
	}
}

func loadYAMLConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func addIptablesRule(hostIP string, rule Rule) {
	gp := strings.Split(rule.GuestPorts, ":")
	guestPorts := strings.Join(gp, "-")
	commands := [][]string{
		{"-I", "FORWARD", "-o", "virbr0", "-p", rule.Protocol, "-d", rule.GuestIP, "--dport", rule.GuestPorts, "-j", "ACCEPT"},
		{"-t", "nat", "-I", "PREROUTING", "-p", rule.Protocol, "-d", hostIP, "--dport", rule.HostPorts, "-j", "DNAT", "--to", fmt.Sprintf("%s:%s", rule.GuestIP, guestPorts)},
	}

	if rule.Allow != "" {
		commands[0] = append(commands[0], "-s", rule.Allow)
	}

	for _, cmd := range commands {
		if err := runCommand("iptables", cmd...); err != nil {
			fmt.Printf("Error adding iptables rule: %v\n", err)
		}
	}
}

func removeIptablesRule(hostIP string, rule Rule) {
	gp := strings.Split(rule.GuestPorts, ":")
	guestPorts := strings.Join(gp, "-")
	commands := [][]string{
		{"-D", "FORWARD", "-o", "virbr0", "-p", rule.Protocol, "-d", rule.GuestIP, "--dport", rule.GuestPorts, "-j", "ACCEPT"},
		{"-t", "nat", "-D", "PREROUTING", "-p", rule.Protocol, "-d", hostIP, "--dport", rule.HostPorts, "-j", "DNAT", "--to", fmt.Sprintf("%s:%s", rule.GuestIP, guestPorts)},
	}

	if rule.Allow != "" {
		commands[0] = append(commands[0], "-s", rule.Allow)
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
