# fortress-csi

The Container Storage Interface ([CSI](https://github.com/container-storage-interface/spec)) Driver for Fortress Block Storage
This driver allows you to use Fortress Block Storage with your container orchestrator.

More information about the CSI and Kubernetes can be found: [CSI Spec](https://github.com/container-storage-interface/spec) and [Kubernetes CSI](https://kubernetes-csi.github.io/docs/example.html)


## Installation
### Requirements

- `--allow-privileged` must be enabled for the API server and kubelet

### Kubernetes secret

In order for the csi to work properly, you will need to deploy a [kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/). To obtain a API key

The `secret.yml` definition is as follows.
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: fortress-csi
  namespace: kube-system
stringData:
  # Replace the api-key with a proper value
  api-key: "FORTRESS_API_KEY"
```

To create this `secret.yml`, you must run the following

```sh
$ kubectl create -f secret.yml            
secret/fortress-csi created
```

### Deploying the CSI

To deploy the latest release of the CSI to your Kubernetes cluster, run the following:

`kubectl apply -f https://raw.githubusercontent.com/mrjosh/fortress-csi/develop/chart/fortress.yaml`


### Validating

The deployment will create a [Storage Class](https://kubernetes.io/docs/concepts/storage/storage-classes/) which will be used to create your volumes

```sh
$ kubectl get storageclass
NAME                               PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
fortress-block-storage (default)   fortress                Delete          Immediate              false                  119m
```

To further validate the CSI, create a [PersistentVolumeClaim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: csi-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: fortress-block-storage
```

Now, take the yaml shown above and create a `pvc.yml` and run:

`kubectl create -f pvc.yml`

You can see that you have a `PersistentVolume` created by your Claim

```sh
$ kubectl get pv
NAME                   CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS     CLAIM             STORAGECLASS             REASON   AGE
pvc-0d87f99b8d5f4419   1Gi        RWO            Delete           Bound      default/csi-pvc   fortress-block-storage            118m
``` 

## Contributing Guidelines
If you are interested in improving or helping with fortress-csi, please feel free to open an issue or PR!
