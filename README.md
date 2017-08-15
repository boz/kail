# kail: kubernetes tail [![Build Status](https://travis-ci.org/boz/kail.svg?branch=master)](https://travis-ci.org/boz/kail)

Kubernetes tail.  Streams logs from all containers of all matched pods.

```sh
# view logs from all pods
$ kail

# all pods named 'x'
$ kail --pod x

# pod named 'y' in namespace 'x'
$ kail --pod x/y

# all pods in namespace 'x' or 'y'
$ kail --ns x --ns y

# pods matching service 'x'
$ kail --svc x

# pods controled by replication controller 'x'
$ kail --rc x

# pods controled by replica set 'x'
$ kail --rc x

# pods on node 'x'
$ kail --node x

# pod with labels 'x=y' and 'a=b'
$ kail --label x=y --label a=b

# pods controlled by replica set 'x', targeted by service 'y', on node 'z'
$ kail --rs x --svc y --node z
```
