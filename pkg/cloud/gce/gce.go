package gce

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/golang/glog"
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

// ListZonesInRegion gets a list of zones in a given region
func (gce *GCEClient) ListZonesInRegion(project, region string) ([]string, error) {
	var zones []string
	r, err := gce.service.Regions.Get(project, region).Do()
	if err != nil {
		return nil, err
	}
	for _, z := range r.Zones {
		zones = append(zones, path.Base(z))
	}
	return zones, nil
}

// GetInstanceGroup returns an instance group by name
func (gce *GCEClient) GetInstanceGroup(project, zone, name string) (*compute.InstanceGroup, error) {
	ig, err := gce.service.InstanceGroups.Get(project, zone, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return nil, nil
			// ig, err = gce.CreateInstanceGroup(project, zone, name)
		} else {
			return nil, err
		}
	}
	return ig, nil
}

// DeleteInstanceGroup returns an instance group by name
func (gce *GCEClient) DeleteInstanceGroup(project, zone, name string) error {
	op, err := gce.service.InstanceGroups.Delete(project, zone, name).Do()
	if err != nil {
		return err
	}
	if err = gce.waitForZoneOp(op, zone); err != nil {
		return err
	}
	return nil
}

// ListInstancesInInstanceGroupForZone lists all the instances in a given instance group for the given zone.
func (gce *GCEClient) ListInstancesInInstanceGroupForZone(name string, zone string) (*compute.InstanceGroupsListInstances, error) {
	ig, err := gce.service.InstanceGroups.ListInstances(
		gce.projectID, zone, name,
		&compute.InstanceGroupsListInstancesRequest{}).Do()
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return ig, nil
}

// AddInstancesToInstanceGroup adds given instance to an Instance Group
func (gce *GCEClient) AddInstancesToInstanceGroup(name, zone string, instances []*compute.InstanceReference) error {
	var err error
	// check instance group exists before trying to add
	ig, err := gce.GetInstanceGroup(gce.projectID, zone, name)
	if err != nil {
		return err
	}
	if ig == nil {
		_, err = gce.CreateInstanceGroup(gce.projectID, zone, name)
		if err != nil {
			return err
		}
	}
	op, err := gce.service.InstanceGroups.AddInstances(
		gce.projectID, zone, name,
		&compute.InstanceGroupsAddInstancesRequest{
			Instances: instances,
		}).Do()
	if err != nil {
		return err
	}
	if err = gce.waitForZoneOp(op, zone); err != nil {
		return err
	}
	return nil
}

// RemoveInstancesFromInstanceGroup adds given instance to an Instance Group
func (gce *GCEClient) RemoveInstancesFromInstanceGroup(name, zone string, instances []*compute.InstanceReference) error {
	op, err := gce.service.InstanceGroups.RemoveInstances(
		gce.projectID, zone, name,
		&compute.InstanceGroupsRemoveInstancesRequest{
			Instances: instances,
		}).Do()
	if err != nil {
		return err
	}
	if err = gce.waitForZoneOp(op, zone); err != nil {
		return err
	}
	return nil
}

// CreateInstanceGroup returns an instance group by name
func (gce *GCEClient) CreateInstanceGroup(project, zone, name string) (*compute.InstanceGroup, error) {
	fmt.Printf("Creating Instance Group %s in %s\n", name, zone)
	ig := &compute.InstanceGroup{
		Name: name,
		Zone: zone,
	}
	op, err := gce.service.InstanceGroups.Insert(project, zone, ig).Do()
	if err != nil {
		return nil, err
	}
	if err = gce.waitForZoneOp(op, zone); err != nil {
		return nil, err
	}
	a, err := gce.GetInstanceGroup(project, zone, name)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ListInstancesInZone returns all instances in a zone
func (gce *GCEClient) ListInstancesInZone(zone string, tags, labels []string) (*compute.InstanceList, error) {
	var filter string
	//fmt.Printf("fetching instances in %s\n", zone)
	list := gce.service.Instances.List(gce.projectID, zone)
	filters := []string{}
	for _, l := range labels {
		s := strings.Split(l, ":")
		filter = "(labels." + s[0] + " eq '" + s[1] + "')" //"job:master"
		filters = append(filters, filter)
	}
	list.Filter(strings.Join(filters, ""))
	// TODO implement tag filter
	return list.Do()
}

// AddInstanceToTargetPool adds instances to the targetpool
func (gce *GCEClient) AddInstanceToTargetPool(region, name string, toAdd []*compute.InstanceReference) error {
	add := &compute.TargetPoolsAddInstanceRequest{Instances: toAdd}
	op, err := gce.service.TargetPools.AddInstance(gce.projectID, region, name, add).Do()
	if err != nil {
		return err
	}
	if err = gce.waitForRegionOp(op, region); err != nil {
		return err
	}
	return nil
}

// DeleteInstanceFromTargetPool deletes instances from the targetpool
func (gce *GCEClient) DeleteInstanceFromTargetPool(region, name string, toDel []*compute.InstanceReference) error {
	del := &compute.TargetPoolsRemoveInstanceRequest{Instances: toDel}
	op, err := gce.service.TargetPools.RemoveInstance(gce.projectID, region, name, del).Do()
	if err != nil {
		return err
	}
	if err = gce.waitForRegionOp(op, region); err != nil {
		return err
	}
	return nil
}

// GetExternalIP confirms that the named External IP exists
func (gce *GCEClient) GetExternalIP(region, name string) (*compute.Address, error) {
	//blerg, err := gce.service.Addresses.List(gce.projectID, region).Do()

	a, err := gce.service.Addresses.Get(gce.projectID, region, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return nil, nil
		} else {
			panic(err)
		}
	}
	return a, nil
}

// CreateExternalIP creates a new external IP to be used with LB
func (gce *GCEClient) CreateExternalIP(region, name string) (*compute.Address, error) {
	address := &compute.Address{
		Name:   name,
		Region: region,
	}
	op, err := gce.service.Addresses.Insert(gce.projectID, region, address).Do()
	if err != nil {
		return nil, err
	}
	if err = gce.waitForRegionOp(op, region); err != nil {
		return nil, err
	}
	a, err := gce.GetExternalIP(region, name)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// RemoveExternalIP deletes the ExternalIP by name.
func (gce *GCEClient) RemoveExternalIP(name, region string) error {
	op, err := gce.service.Addresses.Delete(gce.projectID, region, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, http.StatusNotFound) {
			return nil
		}
		return err
	}
	return gce.waitForRegionOp(op, region)
}

// GetTargetPool gets the list of instances in a targetpool
func (gce *GCEClient) GetTargetPool(region, name string) (*compute.TargetPool, error) {
	tp, err := gce.service.TargetPools.Get(gce.projectID, region, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return nil, nil
		}
	}
	return tp, nil
}

//CreateTargetPool creates a targetpool
func (gce *GCEClient) CreateTargetPool(region, name string, instances []string) (*compute.TargetPool, error) {
	rule := &compute.TargetPool{
		Name:      name,
		Region:    region,
		Instances: instances,
	}
	op, err := gce.service.TargetPools.Insert(gce.projectID, region, rule).Do()
	if err != nil {
		return nil, err
	}
	if err = gce.waitForRegionOp(op, region); err != nil {
		return nil, err
	}
	tp, err := gce.GetTargetPool(region, name)
	if err != nil {
		return nil, err
	}
	return tp, nil
}

// RemoveTargetPool deletes the TargetPool by name.
func (gce *GCEClient) RemoveTargetPool(name, region string) error {
	op, err := gce.service.TargetPools.Delete(gce.projectID, region, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, http.StatusNotFound) {
			return nil
		}
		return err
	}
	return gce.waitForRegionOp(op, region)
}

// GetAvailableZones returns all available zones for this project
func (gce *GCEClient) GetAvailableZones() (*compute.ZoneList, error) {
	return gce.service.Zones.List(gce.projectID).Do()
}

func (gce *GCEClient) GetForwardingRule(region, name string) (*compute.ForwardingRule, error) {
	fr, err := gce.service.ForwardingRules.Get(gce.projectID, region, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return nil, nil
		} else {
			panic(err)
		}
	}
	return fr, err
}

// CreateForwardingRule creates and returns a GlobalForwardingRule that points to the given TargetHttpProxy.
func (gce *GCEClient) CreateForwardingRule(region, name, address, port string) (*compute.ForwardingRule, error) {
	//thp, _ := gce.GetTargetHttpProxy(name)
	t, _ := gce.GetTargetPool(region, name)
	if t == nil {
		return nil, fmt.Errorf("Could not get targetpool %s", name)
	}
	rule := &compute.ForwardingRule{
		Name:       name,
		IPProtocol: "TCP",
		PortRange:  port,
		Target:     t.SelfLink,
		IPAddress:  address,
	}
	op, err := gce.service.ForwardingRules.Insert(gce.projectID, region, rule).Do()
	//fmt.Printf("op %v", op)
	if err != nil {
		return nil, err
	}
	if err = gce.waitForRegionOp(op, region); err != nil {
		return nil, err
	}
	fr, err := gce.GetForwardingRule(region, name)
	if err != nil {
		return nil, err
	}
	return fr, nil
}

// RemoveForwardingRule deletes the GlobalForwardingRule by name.
func (gce *GCEClient) RemoveForwardingRule(name, region string) error {
	op, err := gce.service.ForwardingRules.Delete(gce.projectID, region, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, http.StatusNotFound) {
			return nil
		}
		return err
	}
	return gce.waitForRegionOp(op, region)
}

// makeFirewallObject returns a pre-populated instance of *computeFirewall
func (gce *GCEClient) makeFirewallObject(name, network string, tags []string, port string) (*compute.Firewall, error) {
	firewall := &compute.Firewall{
		Name:         name,
		Description:  "Generated by gcp-lb-tags",
		Network:      makeNetworkURL(gce.projectID, network),
		TargetTags:   tags,
		SourceRanges: []string{"130.211.0.0/22"}, // allow load-balancers alone
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      []string{port},
			},
		},
	}
	return firewall, nil
}

// Backend Services

// GetBackendService retrieves a backend by name.
func (gce *GCEClient) GetBackendService(name string) (*compute.BackendService, error) {
	bs, err := gce.service.BackendServices.Get(gce.projectID, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return nil, nil
		} else {
			return nil, err
		}
	}
	return bs, nil
}

// CreateBackendService creates the given BackendService.
func (gce *GCEClient) CreateBackendService(name string, port string, zones []string) error {

	// prepare backends
	var backends []*compute.Backend
	// one backend (instance group) per zone
	for _, zone := range zones {
		// instance groups have been previously zonified
		ig, _ := gce.GetInstanceGroup(gce.projectID, zone, name)
		if ig != nil {
			backends = append(backends, &compute.Backend{
				Description: zone,
				Group:       ig.SelfLink,
			})
		}
	}
	hc, _ := gce.GetHealthCheck(name, port)

	bsPort, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return err
	}

	//	hc, _ := gce.GetHttpHealthCheck(name)

	// prepare backend service
	bs := &compute.BackendService{
		Backends:     backends,
		HealthChecks: []string{hc.SelfLink},
		Name:         name,
		Port:         bsPort,
		Protocol:     "TCP",
		TimeoutSec:   10, // TODO make configurable
	}
	op, err := gce.service.BackendServices.Insert(gce.projectID, bs).Do()
	if err != nil {
		return err
	}
	return gce.waitForGlobalOp(op)
}

// UpdateBackendService applies the given BackendService as an update to an existing service.
func (gce *GCEClient) UpdateBackendService(name string, port string, zones []string) error {
	bsName := name

	// prepare backends
	var backends []*compute.Backend
	// one backend (instance group) per zone
	for _, zone := range zones {
		// instance groups have been previously zonified
		ig, _ := gce.GetInstanceGroup(gce.projectID, zone, name)
		if ig != nil {
			backends = append(backends, &compute.Backend{
				Description: zone,
				Group:       ig.SelfLink,
			})
		}
	}

	hc, _ := gce.GetHealthCheck(name, port)

	bsPort, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return err
	}

	// need to get the fingerprint of the existing to update
	exist, err := gce.GetBackendService(bsName)
	if err != nil {
		return err
	}
	if exist == nil {
		return fmt.Errorf("backend service does not exist, need to create it in order to update it")
	}

	// prepare backend service
	bs := &compute.BackendService{
		Backends:     backends,
		HealthChecks: []string{hc.SelfLink},
		Name:         bsName,
		Port:         bsPort,
		Protocol:     "TCP",
		Fingerprint:  exist.Fingerprint,
	}

	op, err := gce.service.BackendServices.Update(gce.projectID, bsName, bs).Do()
	if err != nil {
		return err
	}
	return gce.waitForGlobalOp(op)
}

// RemoveBackendService deletes the given BackendService by name.
func (gce *GCEClient) RemoveBackendService(name string) error {
	bsName := name
	op, err := gce.service.BackendServices.Delete(gce.projectID, bsName).Do()
	if err != nil {
		if isHTTPErrorCode(err, http.StatusNotFound) {
			return nil
		}
		return err
	}
	return gce.waitForGlobalOp(op)
}

// healthcheck management
// GetHealthCheck returns the given HttpHealthCheck by name.
func (gce *GCEClient) GetHealthCheck(name, port string) (*compute.HealthCheck, error) {
	hcName := name
	return gce.service.HealthChecks.Get(gce.projectID, hcName).Do()
}

// CreateHealthCheck creates the given HttpHealthCheck.
func (gce *GCEClient) CreateHealthCheck(name string, port string) error {
	hcName := name
	hcPort, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return err
	}
	hc := &compute.HealthCheck{
		Name:           hcName,
		Type:           "TCP",
		TcpHealthCheck: &compute.TCPHealthCheck{Port: hcPort},
	}
	op, err := gce.service.HealthChecks.Insert(gce.projectID, hc).Do()
	if err != nil {
		return err
	}
	return gce.waitForGlobalOp(op)
}

// UpdateHealthCheck applies the given HttpHealthCheck as an update.
func (gce *GCEClient) UpdateHealthCheck(name string, port string) error {
	hcName := name
	hcPort, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return err
	}
	//thc := &compute.TCPHealthCheck{}
	hc := &compute.HealthCheck{
		Name:           hcName,
		Type:           "TCP",
		TcpHealthCheck: &compute.TCPHealthCheck{Port: hcPort},
	}
	op, err := gce.service.HealthChecks.Update(gce.projectID, hcName, hc).Do()
	if err != nil {
		return err
	}
	return gce.waitForGlobalOp(op)
}

// RemoveHealthCheck deletes the given HttpHealthCheck by name.
func (gce *GCEClient) RemoveHealthCheck(name string) error {
	hcName := name
	op, err := gce.service.HealthChecks.Delete(gce.projectID, hcName).Do()
	if err != nil {
		if isHTTPErrorCode(err, http.StatusNotFound) {
			return nil
		}
		return err
	}
	return gce.waitForGlobalOp(op)
}

// Firewall rules management

// CreateFirewall creates a global firewall rule
func (gce *GCEClient) CreateFirewall(name string, network string, port string, tags []string) error {
	fwName := name
	firewall, err := gce.makeFirewallObject(fwName, network, tags, port)
	if err != nil {
		return err
	}
	op, err := gce.service.Firewalls.Insert(gce.projectID, firewall).Do()
	if err != nil && !isHTTPErrorCode(err, http.StatusConflict) {
		return err
	}
	if op != nil {
		err = gce.waitForGlobalOp(op)
		if err != nil && !isHTTPErrorCode(err, http.StatusConflict) {
			return err
		}
	}
	return nil
}

// UpdateFirewall updates a global firewall rule
func (gce *GCEClient) UpdateFirewall(name string, network string, port string, tags []string) error {
	fwName := name
	firewall, err := gce.makeFirewallObject(fwName, network, tags, port)
	if err != nil {
		return err
	}
	op, err := gce.service.Firewalls.Update(gce.projectID, fwName, firewall).Do()
	if err != nil && !isHTTPErrorCode(err, http.StatusConflict) {
		return err
	}
	if op != nil {
		err = gce.waitForGlobalOp(op)
		if err != nil {
			return err
		}
	}
	return nil
}

// RemoveFirewall removes a global firewall rule
func (gce *GCEClient) RemoveFirewall(name string) error {
	fwName := name
	op, err := gce.service.Firewalls.Delete(gce.projectID, fwName).Do()
	if err != nil && isHTTPErrorCode(err, http.StatusNotFound) {
		glog.Infof("Firewall %s already deleted. Continuing to delete other resources.", fwName)
	} else if err != nil {
		glog.Warningf("Failed to delete firewall %s, got error %v", fwName, err)
		return err
	} else {
		if err := gce.waitForGlobalOp(op); err != nil {
			glog.Warningf("Failed waiting for Firewall %s to be deleted.  Got error: %v", fwName, err)
			return err
		}
	}
	return nil
}
