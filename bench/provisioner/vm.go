package provisioner

import (
	"os"
	"path"

	"github.com/grafana/grafana-bench/bench/utils"
	"golang.org/x/crypto/ssh"
)

type VMInstance struct {
	User          string `json:"user"`
	IPAddress     string `json:"ipAddress"`
	ServicePort   string `json:"servicePort"`
	SSHPort       string `json:"sshPort"`
	SSHKeyPath    string `json:"sshKeyPath"`
	SSHKeyPubPath string `json:"sshKeyPubPath"`
}

// ReadVM is called after terraform apply. It reads the VM info from the state
// directory where instanceName is the folder corresponding to that VM.
// e.g. <stateDir>/grafana and <stateDir>/k6
// This assumes a file named, ip_address, sshkey, sshkeypub
// terraform outputs data into. eac
func readVM(stateDir, instanceName string) (*VMInstance, error) {
	ipBytes, err := os.ReadFile(path.Join(stateDir, instanceName, "ip_address"))
	if err != nil {
		return nil, err
	}

	servicePort := ""
	exists, _ := utils.PathExists(path.Join(stateDir, instanceName, "service_port"))
	if exists {
		p, err := os.ReadFile(path.Join(stateDir, instanceName, "service_port"))
		if err != nil {
			return nil, err
		}
		servicePort = string(p)
	}

	userBytes, err := os.ReadFile(path.Join(stateDir, instanceName, "user"))
	if err != nil {
		return nil, err
	}

	return &VMInstance{
		IPAddress:     string(ipBytes),
		ServicePort:   servicePort,
		User:          string(userBytes),
		SSHPort:       "22",
		SSHKeyPath:    path.Join(stateDir, instanceName, "key"),
		SSHKeyPubPath: path.Join(stateDir, instanceName, "key_pub"),
	}, nil
}

// Returns ip_address:servicePort
func (v *VMInstance) ServiceAddress() string {
	return v.IPAddress + ":" + v.ServicePort
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
	return v.IPAddress + ":" + v.SSHPort
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

	// Download the test suite on remote machine
	return session.Run(cmd)
}

// formats map[string]string environment vars into FOO=bar FOO2=bar2 format
func formatEnv(env map[string]string) string {
	envVars := ""
	for k, v := range env {
		envVars += k + "=" + v + " "
	}
	return envVars
}
