package provisioner

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/grafana/grafana-bench/bench/utils"
	"golang.org/x/crypto/ssh"
)

type VMInstance struct {
	User            string `json:"user"`
	Address         string `json:"address"`
	ServicePort     string `json:"servicePort"`
	SSHPort         string `json:"sshPort"`
	SSHKeyPath      string `json:"sshKeyPath"`
	SSHKeyPubPath   string `json:"sshKeyPubPath"`
	StateDir        string `json:"stateDir"`
	InstanceName    string `json:"instanceName"`
	GrafanaUser     string `json:"grafanaUser"`
	GrafanaPassword string `json:"grafanaPassword"`
}

// ReadVM is called after terraform apply. It reads the VM info from the state
// directory where instanceName is the folder corresponding to that VM.
// e.g. <stateDir>/grafana and <stateDir>/k6
// This assumes a file named, ip_address, sshkey, sshkeypub
// terraform outputs data into. eac
func readVM(stateDir, instanceName string) (*VMInstance, error) {
	ipBytes, err := os.ReadFile(vmFilePath(stateDir, instanceName, "ip_address"))
	if err != nil {
		return nil, err
	}

	servicePort := ""
	exists, _ := utils.PathExists(vmFilePath(stateDir, instanceName, "service_port"))
	if exists {
		p, err := os.ReadFile(vmFilePath(stateDir, instanceName, "service_port"))
		if err != nil {
			return nil, err
		}
		servicePort = string(p)
	}

	userBytes, err := os.ReadFile(vmFilePath(stateDir, instanceName, "user"))
	if err != nil {
		return nil, err
	}

	return &VMInstance{
		Address:       string(ipBytes),
		ServicePort:   servicePort,
		User:          string(userBytes),
		SSHPort:       "22",
		SSHKeyPath:    vmFilePath(stateDir, instanceName, "key"),
		SSHKeyPubPath: vmFilePath(stateDir, instanceName, "key_pub"),
		StateDir:      stateDir,
		InstanceName:  instanceName,
	}, nil
}

func vmFilePath(stateDir, instanceName, file string) string {
	return path.Join(stateDir, instanceName, file)
}

// Returns ip_address:servicePort
func (v *VMInstance) ServiceAddress() string {
	return v.Address + ":" + v.ServicePort
}

// Returns https://ip_address:servicePort
func (v *VMInstance) HttpsServiceAddress() string {
	return "https://" + v.ServiceAddress()
}

// Returns http://ip_address:servicePort
func (v *VMInstance) HttpServiceAddress() string {
	return "http://" + v.ServiceAddress()
}

// Return ip_address:sshPort
func (v *VMInstance) SSHAddress() string {
	return v.Address + ":" + v.SSHPort
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

func (v *VMInstance) Run(connection *ssh.Client, cmd string) error {
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
		v.Address,
		vmFilePath(v.StateDir, v.InstanceName, "key"),
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
