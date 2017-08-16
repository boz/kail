for file in _example/{prod,demo,test}.yml; do
  kubectl create -f "$file"
done

./kail

./kail --svc api

./kail --svc prod/api -c nginx

./kail --rs workers --ns test --ns demo

./kail --deploy api -c cache

./kail -l 'app=api,component != worker' -c nginx
