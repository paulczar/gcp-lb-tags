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
	instancesInZone       map[string][]*compute.Instance
	instanceGroups        map[string][]*compute.InstanceWithNamedPorts
	instancesInTargetPool []string
	externalAddress       *compute.Address
}

type loadBalancer struct {
	Address *compute.Address
}

// Cloud interface
type Cloud interface {
	CreateLoadBalancer(cfg *Config) error
	RemoveLoadBalancer(cfg *Config, force bool) error
}

func (c *gceCloud) CreateLoadBalancer(cfg *Config) error {
	var err error
	fmt.Printf("Creating a Loadbalancer for instances with labels:\n - %s\n", strings.Join(cfg.Labels, "\n - "))

	fmt.Printf("--> Updating Target Pool %s\n", cfg.Name)
	err = c.configureInstanceGroups(cfg)
	if err != nil {
		return err
	}

	fmt.Println("--> Creating External Address:")
	c.externalAddress, err = c.client.GetExternalIP(cfg.Region, cfg.Name)
	if err != nil {
		return err
	}
	if c.externalAddress == nil {
		c.externalAddress, err = c.client.CreateExternalIP(cfg.Region, cfg.Name)
		fmt.Printf("====> Created External Address.")
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("====> Used Existing External Address:")
	}
	fmt.Printf(" %s - %s\n", cfg.Name, c.externalAddress.Address)

	// create or update firewall rule
	// try to update first
	fmt.Println("--> Creating Firewall Rule:")
	if err := c.client.UpdateFirewall(cfg.Name, cfg.Network, cfg.Port, cfg.Tags); err != nil {
		// couldn't update most probably because firewall didn't exist
		if err := c.client.CreateFirewall(cfg.Name, cfg.Network, cfg.Port, cfg.Tags); err != nil {
			// couldn't update or create
			return err
		} else {
			fmt.Printf("====> Created Firewall Rule: %s\n", cfg.Name)
		}
	} else {
		fmt.Printf("====> Updated Firewall Rule: %s\n", cfg.Name)
	}

	/** No health checks for tcp based LB
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
	  **/
	/** No backend services for tcp based LB
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
	  **/
	fmt.Println("--> Creating Forwarding Rule:")
	fr, err := c.client.GetForwardingRule(cfg.Region, cfg.Name)
	if err != nil {
		return err
	}
	if fr == nil {
		fmt.Printf("====> Creating Forwarding Rule: %s\n", cfg.Name)
		fr, err = c.client.CreateForwardingRule(cfg.Region, cfg.Name, c.externalAddress.Address, cfg.Port)
		return err
	} else {
		fmt.Printf("====> Using Existing Forwarding Rule: %s\n", cfg.Name)
	}
	fmt.Println("--> Done.")
	return nil
}

func (c *gceCloud) RemoveLoadBalancer(cfg *Config, force bool) error {
	fmt.Printf("Deleting a Loadbalancer for instances with labels %s\n", strings.Join(cfg.Labels, ", "))
	var err error

	fmt.Println("--> Deleting Forwarding Rule")
	if err = c.client.RemoveForwardingRule(cfg.Name, cfg.Region); err != nil {
		return err
	}

	fmt.Printf("--> Delete Target Pool %s\n", cfg.Name)
	if err = c.client.RemoveTargetPool(cfg.Name, cfg.Region); err != nil {
		return err
	}

	fmt.Println("--> Deleting Firewall Rule")
	if err = c.client.RemoveFirewall(cfg.Name); err != nil {
		return err
	}
	if force {
		fmt.Println("--> Deleting External IP")
		if err = c.client.RemoveExternalIP(cfg.Address, cfg.Region); err != nil {
			return err
		}
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

	// get all the instances across all the zones
	instancesInZones := []string{}
	for _, z := range c.zones {
		for _, i := range c.instancesInZone[z] {
			instancesInZones = append(instancesInZones, i.SelfLink)
		}
	}

	// get list of instances in targetpool
	tp, err := c.client.GetTargetPool(cfg.Region, cfg.Name)
	if err != nil {
		return err
	}
	if tp == nil {
		tp, err = c.client.CreateTargetPool(cfg.Region, cfg.Name, instancesInZones)
		if err != nil {
			return err
		}
	} else {
		instancesInTargetPool := tp.Instances
		toAdd := []*compute.InstanceReference{}
		toDel := []*compute.InstanceReference{}
		// create list of Instances in Zone, but not in IG
		for _, i := range instancesInZones {
			found := false
			for _, c := range instancesInTargetPool {
				if i == c {
					found = true
					continue
				}
			}
			if !found {
				fmt.Printf("Need to add %s to TargetPool\n", i)
				toAdd = append(toAdd, &compute.InstanceReference{Instance: i})
			}
		}
		// create list of Instances in IG but not in Zone
		for _, i := range instancesInTargetPool {
			found := false
			for _, c := range instancesInZones {
				if i == c {
					found = true
					continue
				}
			}
			if !found {
				fmt.Printf("Need to remove %s from TargetPool\n", i)
				toDel = append(toDel, &compute.InstanceReference{Instance: i})
			}
		}

		// Add and Delete Instances in TargetPool
		if len(toAdd) > 0 {
			err = c.client.AddInstanceToTargetPool(cfg.Region, cfg.Name, toAdd)
			if err != nil {
				return err
			}
		}
		if len(toDel) > 0 {
			err = c.client.DeleteInstanceFromTargetPool(cfg.Region, cfg.Name, toDel)
			if err != nil {
				return err
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

func (c *gceCloud) AddInstanceToTargetPool(region, name string, toAdd []*compute.InstanceReference) error {
	return c.client.AddInstanceToTargetPool(region, name, toAdd)
}

func (c *gceCloud) DeleteInstanceFromTargetPool(region, name string, toAdd []*compute.InstanceReference) error {
	return c.client.DeleteInstanceFromTargetPool(region, name, toAdd)
}

func (c *gceCloud) CreateForwardingRule(region, name, address string, port string) error {

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
