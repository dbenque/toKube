package deployer

import (
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strconv"

	"strings"

	"github.com/dbenque/toKube/builder"
	"github.com/dbenque/toKube/miniFileServer/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var deploy bool
var deploySuffix string

func init() {
	flag.BoolVar(&deploy, "deploy", false, "To deploy or run")
	flag.StringVar(&deploySuffix, "deploySuffix", "", "Suffix to append to deployment name")
}

func getArgs() []string {
	purgedArgs := []string{}
	if len(os.Args) <= 1 {
		return purgedArgs
	}

	for _, v := range os.Args[1:] {
		param := strings.Split(v, "=")[0]
		switch param {
		case "-deploy", "-deploySuffix":
			continue
		default:
			purgedArgs = append(purgedArgs, v)
		}
	}
	return purgedArgs
}

//AutoDeploy check if the main should be auto-deployed under kubernetes.
//If deployement is requested (flag --deploy) then the program will build and deploy and exit
func AutoDeploy() {

	if deploy {
		fmt.Println("Deployment mode")
		home := os.Getenv("HOME")
		kcliConfig, err := clientcmd.BuildConfigFromFlags("", home+"/.kube/config")
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("Kubernetes host: %s\n", kcliConfig.Host)
		u, err := url.Parse(kcliConfig.Host)
		node, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			//panic(err.Error())
			node = u.Host
		}
		// creates the clientset
		kcli, err := kubernetes.NewForConfig(kcliConfig)
		if err != nil {
			panic(err.Error())
		}
		mfs, err := kcli.CoreV1().Services("default").Get("minifileserver", v1.GetOptions{})
		if err != nil || mfs.Name != "minifileserver" {
			panic("minifileserver service not declared in your kube cluster")
		}
		mfsPort := strconv.Itoa(int(mfs.Spec.Ports[0].NodePort))
		mfsURL := "http://" + net.JoinHostPort(node, mfsPort)

		//use Ingress if exist
		if len(mfs.Status.LoadBalancer.Ingress) > 0 {
			mfsURL = "http://" + mfs.Status.LoadBalancer.Ingress[0].IP
		}

		fmt.Println("Building")
		pwd, _ := os.Getwd()
		_, name := path.Split(pwd)
		if deploySuffix != "" {
			name += "-" + deploySuffix
		}
		b := builder.BuildConfig{Name: name, SourceFolder: "./"}
		b.UseShellEnv()
		binPath, err := b.Build()
		_, binName := path.Split(binPath)
		if err != nil {
			fmt.Printf("Error building: %s", err)
			panic("Error building")
		}

		fmt.Println("Uploading")

		if err := client.PostFile(binPath, mfsURL); err != nil {
			fmt.Printf("Error Uploading: %s", err)
			panic("Error Uploading")
		}

		fmt.Println("Deploying")
		deployment := NewDeploymentFromArgs(strings.ToLower(name))
		deployment.BinaryURL = "http://minifileserver/" + binName
		if err := deployment.Create(kcli); err != nil {
			fmt.Printf("Error Deploying: %s", err)
			panic("Error Deploying")
		}
		if err := deployment.ExposeService(kcli); err != nil {
			fmt.Printf("Error Creating Service: %s", err)
			panic("Error Creating Service")
		}

		fmt.Println("Deployment submitted")
		os.Exit(0)
	}
}

//DeployFolder build and deploy the code in the folder
func DeployFolder(name, folder string, replicas int) {

	fmt.Println("Deployment mode")
	home := os.Getenv("HOME")
	kcliConfig, err := clientcmd.BuildConfigFromFlags("", home+"/.kube/config")
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Kubernetes host: %s\n", kcliConfig.Host)
	u, err := url.Parse(kcliConfig.Host)
	node, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		//panic(err.Error())
		node = u.Host
	}
	// creates the clientset
	kcli, err := kubernetes.NewForConfig(kcliConfig)
	if err != nil {
		panic(err.Error())
	}
	mfs, err := kcli.CoreV1().Services("default").Get("minifileserver", v1.GetOptions{})
	if err != nil || mfs.Name != "minifileserver" {
		panic("minifileserver service not declared in your kube cluster")
	}
	mfsPort := strconv.Itoa(int(mfs.Spec.Ports[0].NodePort))
	mfsURL := "http://" + net.JoinHostPort(node, mfsPort)

	//use Ingress if exist
	if len(mfs.Status.LoadBalancer.Ingress) > 0 {
		mfsURL = "http://" + mfs.Status.LoadBalancer.Ingress[0].IP
	}

	fmt.Println("Building")
	b := builder.BuildConfig{Name: name, SourceFolder: folder}
	b.UseShellEnv()
	binPath, err := b.Build()
	_, binName := path.Split(binPath)
	if err != nil {
		fmt.Printf("Error building: %s", err)
		panic("Error building")
	}

	fmt.Println("Uploading")

	if err := client.PostFile(binPath, mfsURL); err != nil {
		fmt.Printf("Error Uploading: %s", err)
		panic("Error Uploading")
	}

	fmt.Println("Deploying")
	deployment := NewDeploymentFromArgs(strings.ToLower(name))
	deployment.Replicas = replicas
	deployment.BinaryURL = "http://minifileserver/" + binName
	if err := deployment.Create(kcli); err != nil {
		fmt.Printf("Error Deploying: %s", err)
		panic("Error Deploying")
	}
	if err := deployment.ExposeService(kcli); err != nil {
		fmt.Printf("Error Creating Service: %s", err)
		panic("Error Creating Service")
	}
	fmt.Println("Deployment submitted")

}
