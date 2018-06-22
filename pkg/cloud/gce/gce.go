package gce

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// GCEClient is a placeholder for GCE stuff.
type GCEClient struct {
	service    *compute.Service
	projectID  string
	networkURL string
}

// CreateGCECloud creates a new instance of GCECloud.
func CreateGCECloud(project string, network string) (*GCEClient, error) {
	// Use oauth2.NoContext if there isn't a good context to pass in.
	ctx := context.TODO()

	client, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		return nil, err
	}
	svc, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	// TODO validate project and network exist

	return &GCEClient{
		service:   svc,
		projectID: project,
		//networkURL: makeNetworkURL(project, network),
	}, nil
}

func makeNetworkURL(project string, network string) string {
	return fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/networks/%s", project, network)
}

// ListInstancesInZone returns all instances in a zone
func (gce *GCEClient) ListInstancesInZone(zone string) (*compute.InstanceList, error) {
	//fmt.Printf("fetching instances in %s\n", zone)
	return gce.service.Instances.List(gce.projectID, zone).Do()
}

// AddInstanceToTargetPool returns all instances in a zone
func (gce *GCEClient) AddInstanceToTargetPool(region, name string, toAdd []*compute.InstanceReference) (*compute.Operation, error) {
	add := &compute.TargetPoolsAddInstanceRequest{Instances: toAdd}
	return gce.service.TargetPools.AddInstance(gce.projectID, region, name, add).Do()
}

// DeleteInstanceFromTargetPool returns all instances in a zone
func (gce *GCEClient) DeleteInstanceFromTargetPool(region, name string, toDel []*compute.InstanceReference) (*compute.Operation, error) {
	del := &compute.TargetPoolsRemoveInstanceRequest{Instances: toDel}
	return gce.service.TargetPools.RemoveInstance(gce.projectID, region, name, del).Do()
}

// GetTargetPool gets the list of instances in a targetpool
func (gce *GCEClient) GetTargetPool(region, name string) ([]string, error) {
	tp, err := gce.service.TargetPools.Get(gce.projectID, region, name).Do()
	if err != nil {
		log.Fatalf("Target Pool - %v", err)
	}
	return tp.Instances, nil
}
