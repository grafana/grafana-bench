package provisioner

import (
	"os"
	"path"
)

type VMInstance struct {
	IPAddress     string
	ServicePort   string
	SSHKeyPath    string
	SSHKeyPubPath string
}

// ReadVM is called after terraform apply. It reads the VM info from the state
// directory where instanceName is the folder corresponding to that VM.
// e.g. <stateDir>/grafana and <stateDir>/k6
// This assumes a file named, ip_address, sshkey, sshkeypub
// terraform outputs data into. eac
func readVM(stateDir, identifier, instanceName string) (*VMInstance, error) {
	ipBytes, err := os.ReadFile(path.Join(stateDir, instanceName, identifier+"_ip_address"))
	if err != nil {
		return nil, err
	}

	return &VMInstance{
		IPAddress:     string(ipBytes),
		SSHKeyPath:    path.Join(stateDir, instanceName, identifier+"_key"),
		SSHKeyPubPath: path.Join(stateDir, instanceName, identifier+"_key_pub"),
	}, nil
}

// Returns
func (v *VMInstance) ServiceAddress() string {
	return v.IPAddress + ":" + v.Port
}
