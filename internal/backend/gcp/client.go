package gcp

import (
	"context"
	"errors"
	"fmt"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/storage"
)

// Client defines the interface for GCP operations.
// This interface enables dependency injection and mocking for testability,
// mirroring the Docker backend's two-layer pattern (Provider -> Client).
type Client interface {
	// Instance operations
	CreateInstance(ctx context.Context, config instanceConfig) error
	GetInstance(ctx context.Context, project, zone, name string) (*computepb.Instance, error)
	StartInstance(ctx context.Context, project, zone, name string) error
	StopInstance(ctx context.Context, project, zone, name string) error
	ResetInstance(ctx context.Context, project, zone, name string) error
	DeleteInstance(ctx context.Context, project, zone, name string) error

	// Image operations
	CreateImage(ctx context.Context, project string, config imageConfig) error
	GetImage(ctx context.Context, project, name string) (*computepb.Image, error)
	DeleteImage(ctx context.Context, project, name string) error

	// Serial port (for boot/bake diagnostics)
	GetSerialPortOutput(ctx context.Context, project, zone, name string, start int64) (*serialPortOutput, error)

	// Storage operations (GCS for state persistence + log streaming)
	WriteObject(ctx context.Context, bucket, object string, data []byte) error
	ReadObject(ctx context.Context, bucket, object string) ([]byte, error)
	ReadObjectRange(ctx context.Context, bucket, object string, offset int64) ([]byte, error)
	ObjectSize(ctx context.Context, bucket, object string) (int64, error)
	ObjectExists(ctx context.Context, bucket, object string) (bool, error)

	// Close releases all underlying SDK clients.
	Close() error
}

// RealGCPClient implements Client using the official GCP Go SDK.
// It uses Application Default Credentials (ADC) for authentication.
type RealGCPClient struct {
	instances *compute.InstancesClient
	images    *compute.ImagesClient
	storage   *storage.Client
}

// NewRealGCPClient creates a new RealGCPClient with all SDK clients initialized
// using Application Default Credentials (ADC).
//
// ADC supports all standard authentication methods:
//   - GOOGLE_APPLICATION_CREDENTIALS environment variable
//   - gcloud auth application-default login
//   - Service account key files
//   - Workload identity (on GCE, GKE, Cloud Run)
func NewRealGCPClient(ctx context.Context) (*RealGCPClient, error) {
	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create instances client: %w", err)
	}

	imagesClient, err := compute.NewImagesRESTClient(ctx)
	if err != nil {
		_ = instancesClient.Close()

		return nil, fmt.Errorf("failed to create images client: %w", err)
	}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		_ = instancesClient.Close()
		_ = imagesClient.Close()

		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &RealGCPClient{
		instances: instancesClient,
		images:    imagesClient,
		storage:   storageClient,
	}, nil
}

// Close releases all underlying SDK clients.
func (c *RealGCPClient) Close() error {
	var firstErr error

	if err := c.instances.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	if err := c.images.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	if err := c.storage.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}

// CreateInstance creates a GCP Compute Engine VM instance and waits for the
// operation to complete.
func (c *RealGCPClient) CreateInstance(ctx context.Context, config instanceConfig) error {
	networkInterface := &computepb.NetworkInterface{
		Network: strPtr(fmt.Sprintf("global/networks/%s", config.Network)),
	}

	if config.Subnet != "" {
		networkInterface.Subnetwork = strPtr(config.Subnet)
	}

	if config.ExternalIP {
		networkInterface.AccessConfigs = []*computepb.AccessConfig{
			{
				Name: strPtr("External NAT"),
				Type: strPtr("ONE_TO_ONE_NAT"),
			},
		}
	}

	// Build metadata items from config
	var metadataItems []*computepb.Items
	for k, v := range config.Metadata {
		metadataItems = append(metadataItems, &computepb.Items{
			Key:   strPtr(k),
			Value: strPtr(v),
		})
	}

	diskType := config.DiskType
	if diskType == "" {
		diskType = "pd-balanced"
	}

	sourceImage := fmt.Sprintf("projects/%s/global/images/%s", config.ImageProject, config.ImageName)

	instance := &computepb.Instance{
		Name:        strPtr(config.Name),
		MachineType: strPtr(fmt.Sprintf("zones/%s/machineTypes/%s", config.Zone, config.MachineType)),
		Disks: []*computepb.AttachedDisk{
			{
				Boot:       boolPtr(true),
				AutoDelete: boolPtr(true),
				InitializeParams: &computepb.AttachedDiskInitializeParams{
					SourceImage: strPtr(sourceImage),
					DiskSizeGb:  int64Ptr(config.DiskSizeGB),
					DiskType:    strPtr(fmt.Sprintf("zones/%s/diskTypes/%s", config.Zone, diskType)),
				},
			},
		},
		NetworkInterfaces: []*computepb.NetworkInterface{networkInterface},
		Metadata: &computepb.Metadata{
			Items: metadataItems,
		},
		Labels: config.Labels,
	}

	if config.ServiceAccount != "" || len(config.Scopes) > 0 {
		email := config.ServiceAccount
		if email == "" {
			email = "default"
		}

		sa := &computepb.ServiceAccount{
			Email:  strPtr(email),
			Scopes: config.Scopes,
		}

		instance.ServiceAccounts = []*computepb.ServiceAccount{sa}
	}

	op, err := c.instances.Insert(ctx, &computepb.InsertInstanceRequest{
		Project:          config.Project,
		Zone:             config.Zone,
		InstanceResource: instance,
	})
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("instance creation failed: %w", err)
	}

	return nil
}

// GetInstance retrieves information about a VM instance.
func (c *RealGCPClient) GetInstance(ctx context.Context, project, zone, name string) (*computepb.Instance, error) {
	instance, err := c.instances.Get(ctx, &computepb.GetInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return instance, nil
}

// StartInstance starts a stopped VM instance and waits for the operation to complete.
func (c *RealGCPClient) StartInstance(ctx context.Context, project, zone, name string) error {
	op, err := c.instances.Start(ctx, &computepb.StartInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("instance start failed: %w", err)
	}

	return nil
}

// StopInstance stops a running VM instance and waits for the operation to complete.
func (c *RealGCPClient) StopInstance(ctx context.Context, project, zone, name string) error {
	op, err := c.instances.Stop(ctx, &computepb.StopInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("instance stop failed: %w", err)
	}

	return nil
}

// ResetInstance resets a VM instance (hard reboot) and waits for the operation to complete.
func (c *RealGCPClient) ResetInstance(ctx context.Context, project, zone, name string) error {
	op, err := c.instances.Reset(ctx, &computepb.ResetInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to reset instance: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("instance reset failed: %w", err)
	}

	return nil
}

// DeleteInstance deletes a VM instance and waits for the operation to complete.
func (c *RealGCPClient) DeleteInstance(ctx context.Context, project, zone, name string) error {
	op, err := c.instances.Delete(ctx, &computepb.DeleteInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
	})
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("instance deletion failed: %w", err)
	}

	return nil
}

// CreateImage creates a Compute Engine image from a source disk and waits for
// the operation to complete.
func (c *RealGCPClient) CreateImage(ctx context.Context, project string, config imageConfig) error {
	image := &computepb.Image{
		Name:       strPtr(config.Name),
		SourceDisk: strPtr(config.SourceDisk),
		Labels:     config.Labels,
	}

	if config.Description != "" {
		image.Description = strPtr(config.Description)
	}

	op, err := c.images.Insert(ctx, &computepb.InsertImageRequest{
		Project:       project,
		ImageResource: image,
	})
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("image creation failed: %w", err)
	}

	return nil
}

// GetImage retrieves information about a Compute Engine image.
func (c *RealGCPClient) GetImage(ctx context.Context, project, name string) (*computepb.Image, error) {
	image, err := c.images.Get(ctx, &computepb.GetImageRequest{
		Project: project,
		Image:   name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	return image, nil
}

// DeleteImage deletes a Compute Engine image and waits for the operation to complete.
func (c *RealGCPClient) DeleteImage(ctx context.Context, project, name string) error {
	op, err := c.images.Delete(ctx, &computepb.DeleteImageRequest{
		Project: project,
		Image:   name,
	})
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("image deletion failed: %w", err)
	}

	return nil
}

// GetSerialPortOutput reads serial port output from a VM instance.
// The start parameter specifies the byte offset to start reading from (0 for beginning).
func (c *RealGCPClient) GetSerialPortOutput(ctx context.Context, project, zone, name string, start int64) (*serialPortOutput, error) {
	output, err := c.instances.GetSerialPortOutput(ctx, &computepb.GetSerialPortOutputInstanceRequest{
		Project:  project,
		Zone:     zone,
		Instance: name,
		Start:    &start,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get serial port output: %w", err)
	}

	return &serialPortOutput{
		Contents: output.GetContents(),
		Next:     output.GetNext(),
	}, nil
}

// WriteObject writes data to a GCS object, overwriting if it exists.
func (c *RealGCPClient) WriteObject(ctx context.Context, bucket, object string, data []byte) error {
	w := c.storage.Bucket(bucket).Object(object).NewWriter(ctx)

	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return fmt.Errorf("failed to write object: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}

// ReadObject reads the full contents of a GCS object.
func (c *RealGCPClient) ReadObject(ctx context.Context, bucket, object string) ([]byte, error) {
	r, err := c.storage.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	defer func() { _ = r.Close() }()

	data := make([]byte, 0, 1024)
	buf := make([]byte, 4096)

	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}

		if readErr != nil {
			break
		}
	}

	return data, nil
}

// ReadObjectRange reads a GCS object starting from the given byte offset.
func (c *RealGCPClient) ReadObjectRange(ctx context.Context, bucket, object string, offset int64) ([]byte, error) {
	r, err := c.storage.Bucket(bucket).Object(object).NewRangeReader(ctx, offset, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to read object range: %w", err)
	}

	defer func() { _ = r.Close() }()

	data := make([]byte, 0, 1024)
	buf := make([]byte, 4096)

	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}

		if readErr != nil {
			break
		}
	}

	return data, nil
}

// ObjectSize returns the size of a GCS object in bytes.
func (c *RealGCPClient) ObjectSize(ctx context.Context, bucket, object string) (int64, error) {
	attrs, err := c.storage.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get object attrs: %w", err)
	}

	return attrs.Size, nil
}

// ObjectExists checks whether a GCS object exists.
func (c *RealGCPClient) ObjectExists(ctx context.Context, bucket, object string) (bool, error) {
	_, err := c.storage.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

// boolPtr returns a pointer to the given bool.
func boolPtr(b bool) *bool {
	return &b
}

// int64Ptr returns a pointer to the given int64.
func int64Ptr(i int64) *int64 {
	return &i
}
