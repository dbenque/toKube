**MiniFileServer** can stores (POST) and distribute (GET) files.

Files have to be pushed and retrieved one by one.

The image is 3.6Mo

Build:
------

The following script will perform a static build and create a docker image with the MiniFileServer:
```
./docker.sh
```

Deploy:
-------
Publish the image in a registry accessible by your Kubernetes cluster.
For example if you use minikube, do the *** eval $(minikube docker-env) *** before building the docker image.
Adapt the image path inside the mfs.yaml file if needed. 

```
kubectl apply -f mfs.yaml
```

This create both a deployment and a kubernetes service (type nodeport) named "minifileserver"

Usage:
------

***Getting file:***

Example with minikube
```
curl -X GET $(minikube service minifileserver --url)/filename.txt
```

***Posting file:***

Use the function **PostFile** from the package **client** of that project
