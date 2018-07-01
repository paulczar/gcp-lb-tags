package cloud

import (
	"fmt"

	"github.com/paulczar/gcp-lb-tags/pkg/cloud/gce"
	compute "google.golang.org/api/compute/v1"
)

type Config struct {
	Name      string
	Tags      []string
	Labels    []string
	Region    string
	Zones     []string
	ProjectID string
	Network   string
	Ports     []string
	Address   string
}

type instanceGroup struct {
	// map of instances in the instance group
	instances []*compute.Instance
}

type gceCloud struct {
	// GCE client
	client *gce.GCEClient
	config *Config

	// one instance group identifier represents n instance groups, one per available zone
	// e.g. groups := instanceGroups["myIG"]["europe-west1-d"]
	instanceGroups map[string]map[string]*instanceGroup
}

type loadBalancer struct {
	Address *compute.Address
}

// Cloud interface
type Cloud interface {
	ListInstances(zone string) (*compute.InstanceList, error)
	GetTargetPool(region, name string) (*compute.TargetPool, error)
	AddInstanceToTargetPool(region, name string, toAdd []*compute.InstanceReference) error
	DeleteInstanceFromTargetPool(region, name string, toAdd []*compute.InstanceReference) error
	CreateFirewall(name, network string, tags, allowedPorts []string) error
	CreateForwardingRule(region, name, address, target string, ports []string) error
	CreatePublicIP(region, name string) (*compute.Address, error)
	CreateLoadBalancer(cfg *Config) error
	ListZonesInRegion(cfg *Config) ([]string, error)
}

func (c *gceCloud) CreateLoadBalancer(cfg *Config) error {
	// First we need to make sure that an instance group exists
	for _, z := range cfg.Zones {
		// Get a List of instances that match tags
		zoneInstances, err := c.client.ListInstancesInZone(z, cfg.Tags, cfg.Labels)
		if err != nil {
			return err
		}
		for _, ins := range zoneInstances.Items {
			fmt.Printf(" %s", ins.Name)
		}
		fmt.Println()

		if len(zoneInstances.Items) > 0 {
			m := make(map[string]*instanceGroup, 1)
			m[cfg.Name] = &instanceGroup{instances: zoneInstances.Items}
			c.instanceGroups[z] = m
			ig, err := c.client.GetInstanceGroup(cfg.ProjectID, z, cfg.Name)
			if err != nil {
				return err
			}
			if ig == nil {
				c.client.CreateInstanceGroup(cfg.ProjectID, z, cfg.Name)
			}
			fmt.Printf("Instance Group %v", ig.Name)
		}
	}
	return nil
}

func (c *gceCloud) ListZonesInRegion(cfg *Config) ([]string, error) {
	return c.client.ListZonesInRegion(cfg.ProjectID, cfg.Region)
}

func (c *gceCloud) CreatePublicIP(region, name string) (*compute.Address, error) {
	a, err := c.client.GetExternalIP(region, name)
	if err != nil {
		return nil, err
	}
	if a == nil {
		fmt.Printf("No. Will create it. ")
		a, err := c.client.CreateExternalIP(region, name)
		return a, err
	}
	fmt.Printf("Yes. ")
	return a, nil
}

func (c *gceCloud) AddInstanceToTargetPool(region, name string, toAdd []*compute.InstanceReference) error {
	_, err := c.client.AddInstanceToTargetPool(region, name, toAdd)
	return err
}

func (c *gceCloud) DeleteInstanceFromTargetPool(region, name string, toAdd []*compute.InstanceReference) error {
	_, err := c.client.DeleteInstanceFromTargetPool(region, name, toAdd)
	return err
}

func (c *gceCloud) CreateFirewall(name, network string, tags, allowedPorts []string) error {
	exists, _ := c.client.GetFirewall(name)
	if exists == false {
		fmt.Println("No.  creating")
		err := c.client.CreateFirewall(name, network, tags, allowedPorts)
		return err
	}
	fmt.Println("Yes")
	return nil
}

func (c *gceCloud) CreateForwardingRule(region, name, address, target string, ports []string) error {
	fr, err := c.client.GetForwardingRule(region, name)
	if err != nil {
		return err
	}
	if fr == nil {
		fmt.Println("No, creating")
		err = c.client.CreateForwardingRule(region, name, address, target, ports)
		return err
	}
	fmt.Println("Yes")
	return nil
}

func (c *gceCloud) ListInstances(z string) (*compute.InstanceList, error) {
	zoneInstances, err := c.client.ListInstancesInZone(z, []string{}, []string{})
	if err != nil {
		fmt.Printf("err: %v", err)
		return nil, err
	}
	//fmt.Printf("instances: %v", zoneInstances)
	return zoneInstances, nil
}

func (c *gceCloud) GetTargetPool(region, name string) (*compute.TargetPool, error) {
	tp, err := c.client.GetTargetPool(region, name)
	if err != nil {
		fmt.Printf("err: %v", err)
		return nil, err
	}
	return tp, nil
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
		instanceGroups: make(map[string]map[string]*instanceGroup),
	}, nil
}
