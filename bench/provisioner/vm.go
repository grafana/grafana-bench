package provisioner

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/grafana/grafana-bench/bench/utils"
	"golang.org/x/crypto/ssh"
)

type VMInstance struct {
	StateDir     string `json:"stateDir"`
	InstanceName string `json:"instanceName"`

	User          string `json:"user"`
	Host          string `json:"address"`
	SSHPort       string `json:"sshPort"`
	SSHKeyPath    string `json:"sshKeyPath"`
	SSHKeyPubPath string `json:"sshKeyPubPath"`

	ServicePort     string `json:"servicePort"`
	ServiceScheme   string `json:"serviceScheme"`
	ServiceUser     string `json:"grafanaUser"`
	ServicePassword string `json:"grafanaPassword"`
}

// Creates a VM Instance with Grafana Credentials. Does not configure ssh
// credentials.
// Takes a fully qualified address such as https://jefflevinslunch.grafana.net
// and populates the service fields based on the address. If a port is not
// included in the address, it will be determined based on the scheme
func NewReadOnlyGrafanaVM(address, grafanaUser, grafanaPassword string) (*VMInstance, error) {
	scheme, host, port, err := parseServiceAddress(address)
	if err != nil {
		return nil, fmt.Errorf("error parsing grafana uri: %w", err)
	}

	return &VMInstance{
		Host:            host,
		ServicePort:     port,
		ServiceScheme:   scheme,
		ServiceUser:     grafanaUser,
		ServicePassword: grafanaPassword,
	}, nil
}

// parseServiceAddress takes an address such as
// https://jefflevinslunch.grafana.net:3000 and returns scheme, host, port. if
// no port is provided, we assume standard ports based on the url scheme
func parseServiceAddress(address string) (string, string, string, error) {
	u, err := url.Parse(address)
	if err != nil {
		panic(fmt.Errorf("error parsing grafana uri: %w", err))
	}

	// first assume host + port based on scheme
	host := u.Host
	port := ""
	if u.Scheme == "https" {
		port = "443"
	} else if u.Scheme == "http" {
		port = "80"
	} else {
		return "", "", "", fmt.Errorf("unknown scheme: %s. address: %s", u.Scheme, address)
	}

	// check if host includes a port. if it does, split those apart
	if strings.Contains(u.Host, ":") {
		host, port, err = net.SplitHostPort(u.Host)
		if err != nil {
			panic(fmt.Errorf("error parsing grafana uri: %w", err))
		}
	}

	return u.Scheme, host, port, nil
}

// ReadVM is called after terraform apply. It reads the VM info from the state
// directory where instanceName is the folder corresponding to that VM.
// e.g. <stateDir>/grafana and <stateDir>/k6
// This assumes a file named, ip_address, sshkey, sshkeypub
// terraform outputs data into. eac
func readVM(stateDir, instanceName string) (*VMInstance, error) {

	// TODO figure out how to get service port in here

	vmStateDir := path.Join(stateDir, instanceName)

	ipBytes, err := os.ReadFile(path.Join(vmStateDir, "ip_address"))
	if err != nil {
		return nil, err
	}

	servicePort := ""
	exists, _ := utils.PathExists(path.Join(vmStateDir, "service_port"))
	if exists {
		p, err := os.ReadFile(path.Join(vmStateDir, "service_port"))
		if err != nil {
			return nil, err
		}
		servicePort = string(p)
	}

	userBytes, err := os.ReadFile(path.Join(vmStateDir, "user"))
	if err != nil {
		return nil, err
	}

	return &VMInstance{
		Host:          string(ipBytes),
		ServicePort:   servicePort,
		User:          string(userBytes),
		SSHPort:       "22",
		SSHKeyPath:    path.Join(vmStateDir, "key"),
		SSHKeyPubPath: path.Join(vmStateDir, "key_pub"),
		StateDir:      stateDir,
		InstanceName:  instanceName,
	}, nil
}

// Returns ip_address:servicePort
func (v *VMInstance) ServiceAddress() string {
	return v.Host + ":" + v.ServicePort
}

// Returns {scheme}://{ip_address}:{servicePort}
func (v *VMInstance) SchemeServiceAddress() string {
	return fmt.Sprintf("%s://%s:%s", v.ServiceScheme, v.Host, v.ServicePort)
}

// Return ip_address:sshPort
func (v *VMInstance) SSHAddress() string {
	return v.Host + ":" + v.SSHPort
}

// Returns a connection to the vm instance
func (v *VMInstance) Connect() (*ssh.Client, error) {

	// Read the private key file
	pkBytes, err := os.ReadFile(v.SSHKeyPath)
	if err != nil {
		return nil, err
	}

	// Create the SSH private key instance
	privateKey, err := ssh.ParsePrivateKey(pkBytes)
	if err != nil {
		return nil, err
	}

	// SSH client configuration
	config := &ssh.ClientConfig{
		User: v.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(privateKey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Establish the SSH connection
	return ssh.Dial("tcp", v.SSHAddress(), config)
}

func (v *VMInstance) RunCmd(connection *ssh.Client, cmd string) error {
	// Create a new session
	session, err := connection.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Set the session output to os.Stdout
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Download the test suite on remote machine
	return session.Run(cmd)
}

func (v *VMInstance) GetConnectionString() string {
	return fmt.Sprintf("ssh %s@%s -i %s -p %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
		v.User,
		v.Host,
		path.Join(v.StateDir, v.InstanceName, "key"),
		v.SSHPort,
	)
}

// formats map[string]string environment vars into FOO=bar FOO2=bar2 format
func formatEnv(env map[string]string) string {
	envVars := ""
	for k, v := range env {
		envVars += fmt.Sprintf("%s=\"%s\" ", strings.ToUpper(k), strings.TrimSpace(v))
	}
	return envVars
}


// Wait for the grafana instance to start up
func (v *VMInstance) WaitForLiveGrafana(ctx context.Context) error {
	if v.IsLive() {
		return nil
	}

	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		select {
		case <- timer.C:
			if v.IsLive() {
				return nil
			}
		case <- ctx.Done():
			return ctx.Err()
		}
	}
}

// checks if grafana is alive
func (v *VMInstance)IsLive() bool {
	_, err := net.Dial("tcp", v.ServiceAddress())
	return err == nil
}