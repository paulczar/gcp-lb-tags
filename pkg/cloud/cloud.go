package cloud

import (
	"fmt"

	"github.com/paulczar/gcp-lb-tags/pkg/cloud/gce"
	compute "google.golang.org/api/compute/v1"
)

type instanceGroup struct {
	// name of the instance group
	name string
	// map of instances in the instance group
	instances map[string]string
}

type gceCloud struct {
	// GCE client
	client *gce.GCEClient

	// zones available to this project
	zones []string

	// one instance group identifier represents n instance groups, one per available zone
	// e.g. groups := instanceGroups["myIG"]["europe-west1-d"]
	instanceGroups map[string]map[string]*instanceGroup
}

// Cloud interface
type Cloud interface {
	ListInstances(zone string) (*compute.InstanceList, error)
	GetTargetPool(region, name string) ([]string, error)
	AddInstanceToTargetPool(region, name string, toAdd []*compute.InstanceReference) error
	DeleteInstanceFromTargetPool(region, name string, toAdd []*compute.InstanceReference) error
}

func (c *gceCloud) AddInstanceToTargetPool(region, name string, toAdd []*compute.InstanceReference) error {
	_, err := c.client.AddInstanceToTargetPool(region, name, toAdd)
	return err
}

func (c *gceCloud) DeleteInstanceFromTargetPool(region, name string, toAdd []*compute.InstanceReference) error {
	_, err := c.client.DeleteInstanceFromTargetPool(region, name, toAdd)
	return err
}

func (c *gceCloud) ListInstances(z string) (*compute.InstanceList, error) {
	zoneInstances, err := c.client.ListInstancesInZone(z)
	if err != nil {
		fmt.Printf("err: %v", err)
		return nil, err
	}
	//fmt.Printf("instances: %v", zoneInstances)
	return zoneInstances, nil
}

func (c *gceCloud) GetTargetPool(region, name string) ([]string, error) {
	z, err := c.client.GetTargetPool(region, name)
	if err != nil {
		fmt.Printf("err: %v", err)
		return nil, err
	}
	return z, nil
}

// New cloud interface
func New(projectID string, network string, allowedZones []string) (Cloud, error) {
	// try and provision GCE client
	c, err := gce.CreateGCECloud(projectID, network)
	if err != nil {
		return nil, err
	}

	return &gceCloud{
		client:         c,
		zones:          allowedZones,
		instanceGroups: make(map[string]map[string]*instanceGroup),
	}, nil
}
