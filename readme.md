toKube
======

Set of tools for fast development in go with kubernetes.

1) Deploy the minifileserver on your cluster (see mifileserver folder for instructions)
2) In the begining of your main function add the following:
```
	flag.Parse()
	deployer.AutoDeploy()
``` 
3) go run *.go --deploy

This last step will of course do a local build of your app, then:
* it will trigger a static build of your code
* it will upload the binary to the minifileserver
* it will create the ReplicationSet associated to your program. The definition uses initcontainers to
  * Fetch the binary
  * Make the bin runnable

Options
-------

**Container management**
- cpu-limit: Max CPU in milicores (Default:100m)
-	cpu-request: Min CPU in milicores (Default:100m)
-	memory-limit:  Max memory in MB (Default:64M)
-	memory-request: Min memory in MB (Default:64M)
-	namespace: The Kubernetes namespace. (Default:default)
-	replicas: Number of replicas (Default:1)

**Build and image**

By default the code is build in complete static standalone mode. If you prefer it to be linked with glibc, then set the falg *static-build* to false.
In that case your program may not run with default base image (alpine). Select a different base image with flag *base-image*.
