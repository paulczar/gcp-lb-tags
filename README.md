# GCP-LB-LABELS maitains a Google Load Balancer Target Pool based on matching instance labels.

Google LoadBalancing can be a frustrating thing to do, especially when not using managed instance groups.  Different protocols require different google resources (forwarding rule, targetpools, backends, healthchecks, etc) and can be difficult to figure out what to use when.  More-over keeping them up to date when instances are created/removed can be troublesome as well.  This attempts to solve that problem.

Currently only supporting bare TCP protocol (as its sort of the easiest, but also what I needed right away).

## Usage

### CLI

If running on a GCP instance it should auth automatically, if running locally you'll need a .json auth file and point the `GOOGLE_APPLICATION_CREDENTIALS` environment variable at it.

```
$ export GOOGLE_APPLICATION_CREDENTIALS=./google.json
$ ./gcp-lb-tags run --name mydemo --project XXXX --labels job:web --network mydemo
--> Updating instance groups mydemo in zones us-central1-a, us-central1-b, us-central1-c, us-central1-f
--> Updating Public IP
 35.193.26.XX
Created/updated firewall rule with success.
--> Updating Forwarding RuleNo, creating
creating forwarding rule......Done!
```

### Kubernetes

Create a Kubernetes secret from a google auth file:

```
$ kubectl create secret generic boti-google-credentials --from-file=./google.json
```

modify the configmap section of `kubernetes/manifest.yaml` to match your google cloud environment.

Deploy to your kubernetes cluster:

```
$ kubectl apply -f kubernetes/manifest.yaml
```

Here you can see some logs of it in action:

```
$ kc logs -f gcp-lb-tags
2018/06/14 21:39:38 masters are []string{"https://www.googleapis.com/compute/v1/projects/pgtm-XXX/zones/us-central1-a/instances/vm-55fb5210-58bd-4427-6b85-7dc815d13f12"}
2018/06/14 21:39:38 tps are []string{"https://www.googleapis.com/compute/v1/projects/pgtm-XXX/zones/us-central1-a/instances/vm-55fb5210-58bd-4427-6b85-7dc815d13f12"}
2018/06/14 21:39:38 need to add [] and delete [] from targetpool
2018/06/14 21:41:40 masters are []string{"https://www.googleapis.com/compute/v1/projects/pgtm-XXX/zones/us-central1-a/instances/vm-55fb5210-58bd-4427-6b85-7dc815d13f12"}
2018/06/14 21:41:40 tps are []string(nil)
2018/06/14 21:41:40 need to add instance "https://www.googleapis.com/compute/v1/projects/pgtm-XXX/zones/us-central1-a/instances/vm-55fb5210-58bd-4427-6b85-7dc815d13f12" to target pool
2018/06/14 21:41:40 need to add [0xc42029aa00] and delete [] from targetpool
```

## Thanks

Thanks to the author of https://github.com/pires/consul-lb-gce (also Apache licensed) for usable snippets of code for dealing with
various google cloud resources.