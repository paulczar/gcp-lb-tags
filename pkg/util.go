package util

import (
	"fmt"
	"log"
	"sort"

	"github.com/paulczar/gcp-lb-tags/pkg/cloud"
	"github.com/spf13/cobra"
	compute "google.golang.org/api/compute/v1"
)

type Config struct {
	Name      string
	Tags      []string
	Region    string
	Zones     []string
	ProjectID string
	Network   string
}

//var config = &Config{}

//AddorDelInstances Add or delete instances from targetpool
func AddorDelInstances(config *Config) error {
	instancesWithTags, _ := GetInstancesWithTags(config)
	client, err := cloud.New(config.ProjectID, config.Network, config.Zones)
	if err != nil {
		panic(err)
	}
	targetPoolInstances, _ := client.GetTargetPool(config.Region, config.Name)

	toAdd, toDel := DiffInstancesAndTargetpools(instancesWithTags, targetPoolInstances)

	if len(toAdd) > 0 {
		fmt.Printf("adding the following instances to the targetpool: %v", GetInstanceNamesFromList(toAdd))
		err = client.AddInstanceToTargetPool(config.Region, config.Name, toAdd)
		if err != nil {
			log.Fatalf("Could not add instance to target pools: %v", err)
		}
	}

	if len(toDel) > 0 {
		fmt.Printf("deleting the following instances from the targetpool: %v", GetInstanceNamesFromList(toDel))
		err = client.DeleteInstanceFromTargetPool(config.Region, config.Name, toDel)
		if err != nil {
			log.Fatalf("Could not add instance to target pools: %v", err)
		}
	}
	return nil
}

// GetFlagStringSlice can be used to accept multiple argument with flag repetition (e.g. -f arg1,arg2 -f arg3 ...)
func GetFlagStringSlice(cmd *cobra.Command, flag string) []string {
	s, err := cmd.Flags().GetStringSlice(flag)
	if err != nil {
		log.Fatalf("error accessing flag %s for command %s: %v", flag, cmd.Name(), err)
	}
	return s
}

// ExpandZones combines the region with the zone shorthand.
func ExpandZones(c *Config, zones []string) []string {
	region := c.Region
	var ezs []string
	for _, z := range zones {
		ezs = append(ezs, region+"-"+z)
	}
	return ezs
}

// GetInstanceNamesFromList extracts the names of compute instances
func GetInstanceNamesFromList(iList []*compute.InstanceReference) []string {
	var instances []string
	for _, i := range iList {
		instances = append(instances, i.Instance)
	}
	return instances
}

// GetInstancesWithTags lists instances with a specific set of tags
func GetInstancesWithTags(c *Config) ([]*compute.Instance, error) {
	var instances []*compute.Instance
	var foundTags []string

	//fmt.Printf("Looking for instances that contain the tags %v\n", c.Tags)
	client, err := cloud.New(c.ProjectID, c.Network, c.Zones)
	if err != nil {
		panic(err)
	}
	for _, z := range c.Zones {
		res, _ := client.ListInstances(z)
		for _, v := range res.Items {
			for _, t := range v.Tags.Items {
				for _, i := range c.Tags {
					if i == t {
						foundTags = append(foundTags, t)
						continue
					}
				}
			}
			if compareTags(c.Tags, foundTags) {
				instances = append(instances, v)
			}
			foundTags = []string{}
		}
	}
	return instances, nil
}

func compareTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Strings(a)
	sort.Strings(b)
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// DiffInstancesAndTargetpools compares instance lists and determins actions
func DiffInstancesAndTargetpools(instances []*compute.Instance, tps []string) ([]*compute.InstanceReference, []*compute.InstanceReference) {
	var instanceInTp bool
	var tpHasInstances bool
	var toAdd []*compute.InstanceReference
	var toDel []*compute.InstanceReference

	// if master doesn't exist in targetpool, add it.
	for _, i := range instances {
		instanceInTp = false
		for _, t := range tps {
			if i.SelfLink == t {
				instanceInTp = true
				break
			}
		}
		if !instanceInTp {
			log.Printf("need to add instance %#v to target pool", i.SelfLink)
			toAdd = append(toAdd, &compute.InstanceReference{Instance: i.SelfLink})
		}
	}
	// if targetpool instance exists, but not instance, delete it
	for _, t := range tps {
		tpHasInstances = false
		for _, m := range instances {
			if m.SelfLink == t {
				tpHasInstances = true
			}
		}
		if !tpHasInstances {
			log.Printf("need to delete instance %#v from target pool", t)
			toDel = append(toDel, &compute.InstanceReference{Instance: t})
		}
	}
	return toAdd, toDel
}
