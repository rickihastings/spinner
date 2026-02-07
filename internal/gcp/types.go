package gcp

// InstanceConfig holds configuration for creating a GCP Compute Engine VM instance.
type InstanceConfig struct {
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

// ImageConfig holds configuration for creating a GCP Compute Engine image.
type ImageConfig struct {
	// Name is the image name.
	Name string

	// SourceDisk is the full resource URL of the source disk.
	SourceDisk string

	// Description is a human-readable description of the image.
	Description string

	// Labels holds resource labels.
	Labels map[string]string
}

// SerialPortOutput holds output from a VM's serial port.
type SerialPortOutput struct {
	// Contents is the serial port output text.
	Contents string

	// Next is the byte offset for the next read (for polling).
	Next int64
}

// MetricsQuery holds parameters for a Cloud Monitoring time series query.
type MetricsQuery struct {
	// MetricType is the fully-qualified metric type
	// (e.g., "compute.googleapis.com/instance/cpu/utilization").
	MetricType string

	// InstanceName is the VM instance name to filter on.
	InstanceName string

	// Zone is the GCP zone to filter on.
	Zone string

	// IntervalSeconds is how far back to query (in seconds from now).
	IntervalSeconds int64
}

// MetricPoint holds a single data point from a metrics query.
type MetricPoint struct {
	// Value is the metric value (e.g., CPU utilization as 0.0-1.0).
	Value float64
}

// VMStatus represents the status of a GCP VM instance as reported by the API.
type VMStatus string

const (
	// VMStatusProvisioning indicates the VM is being provisioned.
	VMStatusProvisioning VMStatus = "PROVISIONING"

	// VMStatusStaging indicates the VM is being staged.
	VMStatusStaging VMStatus = "STAGING"

	// VMStatusRunning indicates the VM is running.
	VMStatusRunning VMStatus = "RUNNING"

	// VMStatusStopping indicates the VM is being stopped.
	VMStatusStopping VMStatus = "STOPPING"

	// VMStatusStopped indicates the VM is stopped.
	VMStatusStopped VMStatus = "STOPPED"

	// VMStatusSuspending indicates the VM is being suspended.
	VMStatusSuspending VMStatus = "SUSPENDING"

	// VMStatusSuspended indicates the VM is suspended.
	VMStatusSuspended VMStatus = "SUSPENDED"

	// VMStatusTerminated indicates the VM has been terminated.
	VMStatusTerminated VMStatus = "TERMINATED"
)
