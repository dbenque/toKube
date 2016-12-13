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

