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
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// var containing deployment options
// deploy : true if the application should automatically be deployed to current kubernetes cluster
var deploy bool

// deplooySuffix : suffix use for the kubernetes object name to avoid collision in case the same binary need to be deployed several times with different options
var deploySuffix string

// init of that package capture the deployment flags
func init() {
	flag.BoolVar(&deploy, "deploy", false, "To deploy or run")
	flag.StringVar(&deploySuffix, "deploySuffix", "", "Suffix to append to deployment name")
}

// getArgs : capture the args to be passed to the binary when starting the container. It removes the args associated to deployment and keep others.
func getArgs() []string {
	purgedArgs := []string{}
	if len(os.Args) <= 1 {
		return purgedArgs
	}

	for _, v := range os.Args[1:] {
		param := strings.Split(v, "=")[0]
		switch param {
		case "-deploy", "-deploySuffix", "--deploy", "--deploySuffix":
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
		pwd, _ := os.Getwd()
		_, name := path.Split(pwd)
		if deploySuffix != "" {
			name += "-" + deploySuffix
		}
		binPath := build(name, "./")
		uploadAndDeployToKube(name, binPath, 1, getArgs())
		os.Exit(0)
	}
}

//DeployFolder builds and deploys the code in the folder
func DeployFolder(name, folder string, replicas int, args []string) {
	fmt.Println("Deployment mode")
	binPath := build(name, folder)
	uploadAndDeployToKube(name, binPath, replicas, args)
}

func uploadAndDeployToKube(name, binPath string, replicas int, args []string) {
	_, binName := path.Split(binPath)
	kcli, node := getKubeClientAndNode()
	mfsURL := getMinifileserverURL(kcli, node)
	uploadToMinifileServer(binPath, mfsURL)
	deployToKube(name, binName, replicas, kcli, args)
}

func getKubeClientAndNode() (kubernetes.Interface, string) {
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

	return kcli, node
}

func uploadToMinifileServer(binPath, mfsURL string) {
	fmt.Println("Uploading")

	if err := client.PostFile(binPath, mfsURL); err != nil {
		fmt.Printf("Error Uploading: %s", err)
		panic("Error Uploading")
	}
}

func getMinifileserverURL(kcli kubernetes.Interface, kubeNode string) string {
	mfs, err := kcli.CoreV1().Services("default").Get("minifileserver", v1.GetOptions{})
	if err != nil || mfs.Name != "minifileserver" {
		panic("minifileserver service not declared in your kube cluster")
	}
	mfsPort := strconv.Itoa(int(mfs.Spec.Ports[0].NodePort))
	mfsURL := "http://" + net.JoinHostPort(kubeNode, mfsPort)

	//use Ingress if exist
	if len(mfs.Status.LoadBalancer.Ingress) > 0 {
		mfsURL = "http://" + mfs.Status.LoadBalancer.Ingress[0].IP
	}
	return mfsURL
}

func build(name string, sourceFolder string) (binFullPath string) {
	fmt.Println("Building")

	b := builder.BuildConfig{Name: name, SourceFolder: sourceFolder}
	b.UseShellEnv()
	var err error
	binFullPath, err = b.Build()
	if err != nil {
		fmt.Printf("Error building: %s", err)
		panic("Error building")
	}
	return
}

func deployToKube(name, binName string, replicas int, kcli kubernetes.Interface, args []string) {
	fmt.Println("Deploying")
	deployment := NewDeploymentFromArgs(strings.ToLower(name))
	deployment.Args = args
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
