# kail: kubernetes tail (wip)

```sh
# view logs from all pods
$ kail

# all pods named 'x'
$ kail --pod x

# pod named 'y' in namespace 'x'
$ kail --pod x/y

# pod with labels 'x=y' and 'a=b'
$ kail --label 'x=y'

# all pods in namespace 'x' or 'y'
$ kail --ns x

# pods controled by replication controller 'x'
$ kail --rc x

# pods matching service 'x'
$ kail --svc x

# pods on node 'x'
$ kail --node x
```
