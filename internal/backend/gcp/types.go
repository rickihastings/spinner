package gcp

// instanceConfig holds configuration for creating a GCP Compute Engine VM instance.
type instanceConfig struct {
	// Name is the VM instance name (must match GCP naming: lowercase, max 63 chars).
	Name string

	// Project is the GCP project ID.
	Project string

	// Zone is the GCP zone (e.g., "us-central1-a").
	Zone string

	// MachineType is the VM machine type (e.g., "e2-standard-2").
	MachineType string

	// ImageProject is the project containing the source image.
	ImageProject string

	// ImageName is the custom image name to boot from.
	ImageName string

	// DiskSizeGB is the boot disk size in GB.
	DiskSizeGB int64

	// DiskType is the boot disk type (e.g., "pd-balanced", "pd-ssd", "pd-standard").
	DiskType string

	// Network is the VPC network name (default: "default").
	Network string

	// Subnet is the subnetwork name (optional).
	Subnet string

	// ExternalIP controls whether to assign an ephemeral external IP.
	ExternalIP bool

	// Metadata holds instance metadata key-value pairs (startup script, env vars).
	Metadata map[string]string

	// Labels holds resource labels for identification and cost tracking.
	Labels map[string]string

	// ServiceAccount is the service account email (optional, uses default).
	ServiceAccount string

	// Scopes is the list of OAuth scopes for the service account.
	Scopes []string
}

// imageConfig holds configuration for creating a GCP Compute Engine image.
type imageConfig struct {
	// Name is the image name.
	Name string

	// SourceDisk is the full resource URL of the source disk.
	SourceDisk string

	// Description is a human-readable description of the image.
	Description string

	// Labels holds resource labels.
	Labels map[string]string
}

// serialPortOutput holds output from a VM's serial port.
type serialPortOutput struct {
	// Contents is the serial port output text.
	Contents string

	// Next is the byte offset for the next read (for polling).
	Next int64
}

// GCPInstance represents a Compute Engine VM instance.
type GCPInstance struct {
	Name              string                `json:"name"`
	Status            string                `json:"status"`
	MachineType       string                `json:"machineType"`
	Zone              string                `json:"zone"`
	Disks             []GCPDisk             `json:"disks"`
	NetworkInterfaces []GCPNetworkInterface `json:"networkInterfaces"`
	Metadata          *GCPMetadata          `json:"metadata"`
	Labels            map[string]string     `json:"labels"`
	ServiceAccounts   []GCPServiceAccount   `json:"serviceAccounts"`
}

// GCPDisk represents an attached disk on a Compute Engine instance.
type GCPDisk struct {
	Source     string `json:"source"`
	Boot       bool   `json:"boot"`
	AutoDelete bool   `json:"autoDelete"`
}

// GCPNetworkInterface represents a network interface on a Compute Engine instance.
type GCPNetworkInterface struct {
	Network       string            `json:"network"`
	Subnetwork    string            `json:"subnetwork"`
	AccessConfigs []GCPAccessConfig `json:"accessConfigs"`
}

// GCPAccessConfig represents an access configuration for a network interface.
type GCPAccessConfig struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	NatIP string `json:"natIP"`
}

// GCPMetadata represents instance metadata with a fingerprint for consistency.
type GCPMetadata struct {
	Fingerprint string            `json:"fingerprint"`
	Items       []GCPMetadataItem `json:"items"`
}

// GCPMetadataItem represents a single metadata key-value pair.
type GCPMetadataItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GCPServiceAccount represents a service account attached to an instance.
type GCPServiceAccount struct {
	Email  string   `json:"email"`
	Scopes []string `json:"scopes"`
}

// GCPImage represents a Compute Engine image.
type GCPImage struct {
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	SourceDisk  string            `json:"sourceDisk"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
}

// vmStatus represents the status of a GCP VM instance as reported by the API.
type vmStatus string

const (
	// vmStatusProvisioning indicates the VM is being provisioned.
	vmStatusProvisioning vmStatus = "PROVISIONING"

	// vmStatusStaging indicates the VM is being staged.
	vmStatusStaging vmStatus = "STAGING"

	// vmStatusRunning indicates the VM is running.
	vmStatusRunning vmStatus = "RUNNING"

	// vmStatusStopping indicates the VM is being stopped.
	vmStatusStopping vmStatus = "STOPPING"

	// vmStatusStopped indicates the VM is stopped.
	vmStatusStopped vmStatus = "STOPPED"

	// vmStatusSuspending indicates the VM is being suspended.
	vmStatusSuspending vmStatus = "SUSPENDING"

	// vmStatusSuspended indicates the VM is suspended.
	vmStatusSuspended vmStatus = "SUSPENDED"

	// vmStatusTerminated indicates the VM has been terminated.
	vmStatusTerminated vmStatus = "TERMINATED"
)
