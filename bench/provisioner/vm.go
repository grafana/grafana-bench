package provisioner

import (
	"os"
	"path"

	"golang.org/x/crypto/ssh"
)

type VMInstance struct {
	User          string
	IPAddress     string
	ServicePort   string
	SSHPort       string
	SSHKeyPath    string
	SSHKeyPubPath string
}

// ReadVM is called after terraform apply. It reads the VM info from the state
// directory where instanceName is the folder corresponding to that VM.
// e.g. <stateDir>/grafana and <stateDir>/k6
// This assumes a file named, ip_address, sshkey, sshkeypub
// terraform outputs data into. eac
func readVM(stateDir, identifier, instanceName string) (*VMInstance, error) {
	ipBytes, err := os.ReadFile(path.Join(stateDir, instanceName, "ip_address"))
	if err != nil {
		return nil, err
	}

	return &VMInstance{
		IPAddress:     string(ipBytes),
		SSHKeyPath:    path.Join(stateDir, instanceName, "ssh_key"),
		SSHKeyPubPath: path.Join(stateDir, instanceName, "ssh_key_pub"),
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
