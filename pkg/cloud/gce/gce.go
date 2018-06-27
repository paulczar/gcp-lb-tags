package gce

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	operationPollInterval        = 3 * time.Second
	operationPollTimeoutDuration = 30 * time.Minute
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

// GetTargetPool gets the list of instances in a targetpool
func (gce *GCEClient) GetTargetPool(region, name string) (*compute.TargetPool, error) {
	tp, err := gce.service.TargetPools.Get(gce.projectID, region, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			tp, err = gce.CreateTargetPool(region, name)
			if err != nil {
				return nil, err
			} else {
				return nil, nil
			}
		} else {
			panic(err)
		}
	}
	return tp, nil
}

func (gce *GCEClient) CreateTargetPool(region, name string) (*compute.TargetPool, error) {
	rule := &compute.TargetPool{
		Name:   name,
		Region: region,
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

// GetAvailableZones returns all available zones for this project
func (gce *GCEClient) GetAvailableZones() (*compute.ZoneList, error) {
	return gce.service.Zones.List(gce.projectID).Do()
}

// CreateFirewall creates a global firewall rule
func (gce *GCEClient) CreateFirewall(name, network string, tags, allowedPorts []string) error {
	firewall, err := gce.makeFirewallObject(name, network, tags, allowedPorts)
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

// GetFirewall checks if Firewall exists
func (gce *GCEClient) GetFirewall(name string) (bool, error) {
	_, err := gce.service.Firewalls.Get(gce.projectID, name).Do()
	if err != nil {
		if isHTTPErrorCode(err, 404) {
			return false, nil
		} else {
			panic(err)
		}
	}
	//fmt.Printf("firewall %v\n", fw)
	return true, nil
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
func (gce *GCEClient) CreateForwardingRule(region, name, address, target string, ports []string) error {
	//thp, _ := gce.GetTargetHttpProxy(name)
	rule := &compute.ForwardingRule{
		Name:       name,
		IPProtocol: "TCP",
		PortRange:  strings.Join(ports, ","),
		Target:     target,
		IPAddress:  address,
	}
	fmt.Printf("creating forwarding rule...")
	op, err := gce.service.ForwardingRules.Insert(gce.projectID, region, rule).Do()
	//fmt.Printf("op %v", op)
	if err != nil {
		return err
	}
	if err = gce.waitForRegionOp(op, region); err != nil {
		return err
	}
	fmt.Printf("...Done!")
	return nil
}

func (gce *GCEClient) waitForGlobalOp(op *compute.Operation) error {
	return waitForOp(op, func(operationName string) (*compute.Operation, error) {
		return gce.service.GlobalOperations.Get(gce.projectID, operationName).Do()
	})
}

func (gce *GCEClient) waitForRegionOp(op *compute.Operation, region string) error {
	return waitForOp(op, func(operationName string) (*compute.Operation, error) {
		return gce.service.RegionOperations.Get(gce.projectID, region, operationName).Do()
	})
}

func isHTTPErrorCode(err error, code int) bool {
	apiErr, ok := err.(*googleapi.Error)
	return ok && apiErr.Code == code
}

func waitForOp(op *compute.Operation, getOperation func(operationName string) (*compute.Operation, error)) error {
	if op == nil {
		return fmt.Errorf("operation must not be nil")
	}

	if opIsDone(op) {
		return getErrorFromOp(op)
	}

	opName := op.Name
	return wait.Poll(operationPollInterval, operationPollTimeoutDuration, func() (bool, error) {
		pollOp, err := getOperation(opName)
		if err != nil {
			fmt.Printf("GCE poll operation failed: %v\n", err)
		}
		return opIsDone(pollOp), getErrorFromOp(pollOp)
	})
}

func opIsDone(op *compute.Operation) bool {
	return op != nil && op.Status == "DONE"
}

func getErrorFromOp(op *compute.Operation) error {
	if op != nil && op.Error != nil && len(op.Error.Errors) > 0 {
		err := &googleapi.Error{
			Code:    int(op.HttpErrorStatusCode),
			Message: op.Error.Errors[0].Message,
		}
		return fmt.Errorf("GCE operation failed: %v", err)
	}

	return nil
}

// makeFirewallObject returns a pre-populated instance of *computeFirewall
func (gce *GCEClient) makeFirewallObject(name, network string, tags, allowedPorts []string) (*compute.Firewall, error) {
	firewall := &compute.Firewall{
		Name:         name,
		Description:  "Generated by gcp-lb-tags",
		Network:      makeNetworkURL(gce.projectID, network),
		TargetTags:   tags,
		SourceRanges: []string{"0.0.0.0/0"}, // allow load-balancers alone
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      allowedPorts,
			},
		},
	}
	return firewall, nil
}
