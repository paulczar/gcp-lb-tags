# GCP-LB-TAGS maitains a Google Load Balancer Target Pool based on matching instance tags.

Given an existing Load Balancer, will keep it's target pool updated to contain all instances that match a set of tags in your given region and zones.

## Usage

### CLI

If running on a GCP instance it should auth automatically, if running locally you'll need a .json auth file and point the `GOOGLE_APPLICATION_CREDENTIALS` environment variable at it.

```
$ export GOOGLE_APPLICATION_CREDENTIALS=./google.json
$ ./gcp-lb-tags run --name mydemo-pks-cluster1 --project pgtm-pczarkowski --tags master,mydemo --network mydemo
Ensuring that TargetPool mydemo-pks-cluster1 contains instances in us-central1 with [master mydemo]
```

#### Kubernetes

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