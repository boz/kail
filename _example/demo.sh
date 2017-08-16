#!/bin/sh

# ./kail
# ./kail --svc api
# ./kail --svc prod/api -c nginx
# ./kail --rs workers --ns test --ns demo
# ./kail --deploy api -c cache
# ./kail -l 'app=api,component != worker' -c nginx

start() {
  for file in $(dirname $0)/{prod,demo,test}.yml; do
    kubectl create -f "$file"
  done
}

stop() {
  for file in $(dirname $0)/{prod,demo,test}.yml; do
    kubectl delete -f "$file"
  done
}

case "$1" in
  start) start;;
  stop) stop;;
esac

