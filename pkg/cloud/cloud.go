package cloud

import (
	"fmt"
	"strings"

	"github.com/paulczar/gcp-lb-tags/pkg/cloud/gce"
	compute "google.golang.org/api/compute/v1"
)

type Config struct {
	Name      string
	Tags      []string
	Labels    []string
	Region    string
	ProjectID string
	Network   string
	Ports     []string
	Port      string
	Address   string
	Zones     []string
}

type gceCloud struct {
	// GCE client
	client *gce.GCEClient

	zones []string
	// instances per zone
	// e.g. groups := instances["europe-west1-d"]
	instancesInZone map[string][]*compute.Instance
	instanceGroups  map[string][]*compute.InstanceWithNamedPorts
}

type loadBalancer struct {
	Address *compute.Address
}

// Cloud interface
type Cloud interface {
	CreateLoadBalancer(cfg *Config) error
}

func (c *gceCloud) CreateLoadBalancer(cfg *Config) error {
	var err error
	fmt.Printf("Creating a Loadbalancer for instances with labels %s\n", strings.Join(cfg.Labels, ", "))
	fmt.Printf("--> Updating instance groups %s in zones %s\n", cfg.Name, strings.Join(c.zones, ", "))
	err = c.configureInstanceGroups(cfg)
	if err != nil {
		return err
	}

	fmt.Println("--> Updating Public IP")
	address, err := c.CreatePublicIP(cfg.Region, cfg.Address)
	if err != nil {
		panic(err)
	}
	fmt.Printf(" %s\n", address.Address)

	// create or update firewall rule
	// try to update first
	if err := c.client.UpdateFirewall(cfg.Name, cfg.Network, cfg.Port, cfg.Tags); err != nil {
		// couldn't update most probably because firewall didn't exist
		if err := c.client.CreateFirewall(cfg.Name, cfg.Network, cfg.Port, cfg.Tags); err != nil {
			// couldn't update or create
			return err
		}
	}
	fmt.Println("Created/updated firewall rule with success.")

	// ensure health checks are set up
	fmt.Println("--> Updating Health Check")
	for _, port := range cfg.Ports {
		if err := c.client.UpdateHealthCheck(cfg.Name, port); err != nil {
			// couldn't update most probably because health-check didn't exist
			if err := c.client.CreateHealthCheck(cfg.Name, port); err != nil {
				// couldn't update or create
				return err
			} else {
				fmt.Printf("====> Created Health check for port %s\n", port)
			}
		} else {
			fmt.Printf("====> Updated Health check for port %s\n", port)
		}
	}

	fmt.Println("--> Updating Backend Services")
	// create or update backend service, only for allowed zones
	// try to update first
	if err := c.client.UpdateBackendService(cfg.Name, cfg.Port, c.zones); err != nil {
		// couldn't update most probably because backend service didn't exist
		//return err
		if err := c.client.CreateBackendService(cfg.Name, cfg.Port, c.zones); err != nil {
			// couldn't update or create
			return err
		}
	}
	fmt.Println("====> Created/updated backend service with success.")

	fmt.Printf("--> Updating Forwarding Rule")
	err = c.CreateForwardingRule(cfg.Region, cfg.Name, address.SelfLink, cfg.Port)
	if err != nil {
		panic(err)
	}

	return nil
}

func (c *gceCloud) configureInstanceGroups(cfg *Config) error {
	var err error

	// get list of instances in the zone
	err = c.listInstancesPerZone(cfg)
	if err != nil {
		return err
	}
	// get list if instances in the instance group in the zone
	err = c.listInstancesInInstanceGroups(cfg)
	if err != nil {
		return err
	}
	//fmt.Printf("instances in zone: %#v", c.instancesInZone)
	for _, z := range c.zones {
		//fmt.Printf("Compare %v to %v\n", c.instanceGroups[z], c.instancesInZone[z])
		instancesInZone := []string{}
		instancesInIG := []string{}
		toAdd := []*compute.InstanceReference{}
		toDel := []*compute.InstanceReference{}
		fmt.Printf("====> Looking at instances in zone %s\n", z)
		if len(c.instancesInZone[z]) == 0 && len(c.instanceGroups[z]) == 0 {
			fmt.Printf("No instances found in zone, do nothing\n")
			continue
		}
		for _, i := range c.instancesInZone[z] {
			instancesInZone = append(instancesInZone, i.SelfLink)
		}
		for _, i := range c.instanceGroups[z] {
			instancesInIG = append(instancesInIG, i.Instance)
		}

		// create list of Instances in Zone, but not in IG
		for _, i := range instancesInZone {
			found := false
			for _, c := range instancesInIG {
				if i == c {
					found = true
					continue
				}
			}
			if !found {
				fmt.Printf("Need to add %s to Instance Group\n", i)
				toAdd = append(toAdd, &compute.InstanceReference{Instance: i})
			}
		}

		// create list of Instances in IG but not in Zone
		for _, i := range instancesInIG {
			found := false
			for _, c := range instancesInZone {
				if i == c {
					found = true
					continue
				}
			}
			if !found {
				fmt.Printf("Need to remove %s from Instance Group\n", i)
				toDel = append(toDel, &compute.InstanceReference{Instance: i})
			}
		}

		// Add and Delete Instances in Instance Groups
		if len(toAdd) > 0 {
			err = c.client.AddInstancesToInstanceGroup(cfg.Name, z, toAdd)
			if err != nil {
				return err
			}
		}
		if len(toDel) > 0 {
			err = c.client.RemoveInstancesFromInstanceGroup(cfg.Name, z, toDel)
			if err != nil {
				return err
			}
		}
	}
	// update list of instances in the instance groups
	err = c.listInstancesInInstanceGroups(cfg)
	if err != nil {
		return err
	}
	for z, i := range c.instanceGroups {
		if len(i) == 0 {
			ig, err := c.client.GetInstanceGroup(cfg.ProjectID, z, cfg.Name)
			if err != nil {
				return err
			}
			if ig != nil {
				fmt.Printf("Delete instance group %s in zone %s as it is empty.\n", cfg.Name, z)
				c.client.DeleteInstanceGroup(cfg.ProjectID, z, cfg.Name)
			}
		}
	}
	return nil
}

func (c *gceCloud) listInstancesPerZone(cfg *Config) error {
	// First we need to make sure that an instance group exists
	for _, z := range c.zones {
		// Get a List of instances that match tags
		zi, err := c.client.ListInstancesInZone(z, cfg.Tags, cfg.Labels)
		if err != nil {
			return err
		}
		c.instancesInZone[z] = zi.Items
	}
	return nil
}

func (c *gceCloud) listInstancesInInstanceGroups(cfg *Config) error {
	for _, z := range c.zones {
		// fetch instance group for the zone
		ig, err := c.client.GetInstanceGroup(cfg.ProjectID, z, cfg.Name)
		if err != nil {
			return err
		}

		// if instance group doesn't exist move to next zone
		if ig == nil {
			c.instanceGroups[z] = nil
			continue
		}

		// Fetch list of instances in instance group for the zone
		igl, err := c.client.ListInstancesInInstanceGroupForZone(cfg.Name, z)
		if igl != nil {
			c.instanceGroups[z] = igl.Items
			if err != nil {
				return err
			}
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
		//		fmt.Printf("No. Will create it. ")
		a, err := c.client.CreateExternalIP(region, name)
		return a, err
	}
	//	fmt.Printf("Yes. ")
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

func (c *gceCloud) CreateForwardingRule(region, name, address string, port string) error {
	fr, err := c.client.GetForwardingRule(region, name)
	if err != nil {
		return err
	}
	if fr == nil {
		fmt.Println("No, creating")
		err = c.client.CreateForwardingRule(region, name, address, port)
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
func New(projectID string, network string, region string) (Cloud, error) {
	// try and provision GCE client
	c, err := gce.CreateGCECloud(projectID, network)
	if err != nil {
		return nil, err
	}
	zones, err := c.ListZonesInRegion(projectID, region)
	if err != nil {
		return nil, err
	}

	return &gceCloud{
		client:          c,
		zones:           zones,
		instancesInZone: make(map[string][]*compute.Instance),
		instanceGroups:  make(map[string][]*compute.InstanceWithNamedPorts),
	}, nil
}
