package deployer

import (
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/resource"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
)

var (
	cpuLimit      string
	cpuRequest    string
	memoryLimit   string
	memoryRequest string
	namespace     string
	baseImage     string
	replicas      int
)

func init() {
	flag.StringVar(&cpuLimit, "cpu-limit", "100m", "Max CPU in milicores")
	flag.StringVar(&cpuRequest, "cpu-request", "100m", "Min CPU in milicores")
	flag.StringVar(&memoryLimit, "memory-limit", "64M", "Max memory in MB")
	flag.StringVar(&memoryRequest, "memory-request", "64M", "Min memory in MB")
	flag.StringVar(&namespace, "namespace", "default", "The Kubernetes namespace.")
	flag.IntVar(&replicas, "replicas", 1, "Number of replicas")
	flag.StringVar(&baseImage, "base-image", "alpine:3.4", "Base image to run the container")
}

//Deployment contains configuration for the deployment
type Deployment struct {
	Annotations   map[string]string
	Args          []string
	Env           map[string]string
	BinaryURL     string
	cpuRequest    string
	cpuLimit      string
	memoryRequest string
	memoryLimit   string
	Name          string
	Namespace     string
	Replicas      int
	Labels        map[string]string
}

//NewDeploymentFromArgs prepare a deployment based on the command line parameters
func NewDeploymentFromArgs(name string) *Deployment {
	return &Deployment{
		cpuRequest:    cpuRequest,
		cpuLimit:      cpuLimit,
		memoryRequest: memoryRequest,
		memoryLimit:   memoryLimit,
		Replicas:      replicas,
		Namespace:     namespace,
		Annotations:   map[string]string{},
		Labels:        map[string]string{},
		Env:           map[string]string{},
		Args:          []string{},
		Name:          name,
	}
}

//Create the a deployment
func (d *Deployment) Create(kclientset kubernetes.Interface) error {

	volumes := make([]v1.Volume, 0)
	volumes = append(volumes, v1.Volume{
		Name:         "bin",
		VolumeSource: v1.VolumeSource{},
	})

	volumeMounts := make([]v1.VolumeMount, 0)
	volumeMounts = append(volumeMounts, v1.VolumeMount{
		Name:      "bin",
		MountPath: "/opt/bin",
	})

	container := v1.Container{}
	container.Args = d.Args
	container.Command = []string{filepath.Join("/opt/bin", d.Name)}
	container.Image = baseImage
	container.Name = d.Name
	container.VolumeMounts = volumeMounts
	container.Ports = []v1.ContainerPort{v1.ContainerPort{ContainerPort: 9102}} // for prometheus

	resourceLimits := make(v1.ResourceList)
	if d.cpuLimit != "" {
		resourceLimits[v1.ResourceCPU] = resource.MustParse(d.cpuLimit)
	}
	if d.memoryLimit != "" {
		resourceLimits[v1.ResourceMemory] = resource.MustParse(d.memoryLimit)
	}

	resourceRequests := make(v1.ResourceList)
	if d.cpuRequest != "" {
		resourceRequests[v1.ResourceCPU] = resource.MustParse(d.cpuRequest)
	}
	if d.memoryRequest != "" {
		resourceRequests[v1.ResourceMemory] = resource.MustParse(d.memoryRequest)
	}

	if len(resourceLimits) > 0 {
		container.Resources.Limits = resourceLimits
	}
	if len(resourceRequests) > 0 {
		container.Resources.Requests = resourceRequests
	}

	if len(d.Env) > 0 {
		env := make([]v1.EnvVar, 0)
		for name, value := range d.Env {
			env = append(env, v1.EnvVar{Name: name, Value: value})
		}
		container.Env = env
	}

	annotations := d.Annotations

	binaryPath := filepath.Join("/opt/bin", d.Name)
	initContainers := []v1.Container{
		v1.Container{
			Name:    "install",
			Image:   "alpine:3.4",
			Command: []string{"wget", "-O", binaryPath, d.BinaryURL},
			VolumeMounts: []v1.VolumeMount{
				v1.VolumeMount{
					Name:      "bin",
					MountPath: "/opt/bin",
				},
			},
		},
		v1.Container{
			Name:    "configure",
			Image:   "alpine:3.4",
			Command: []string{"chmod", "+x", binaryPath},
			VolumeMounts: []v1.VolumeMount{
				v1.VolumeMount{
					Name:      "bin",
					MountPath: "/opt/bin",
				},
			},
		},
	}

	ic, err := json.MarshalIndent(&initContainers, "", " ")
	if err != nil {
		return err
	}
	annotations["pod.beta.kubernetes.io/init-containers"] = string(ic)
	annotations["prometheus.io/scrape"] = "true"

	d.Labels["run"] = d.Name

	var replica int32
	replica = int32(d.Replicas)

	krs := v1beta1.ReplicaSet{}
	krs.APIVersion = "extensions/v1beta1"
	krs.Kind = "ReplicaSet"
	krs.Name = d.Name
	krs.Namespace = d.Namespace
	krs.Spec = v1beta1.ReplicaSetSpec{}
	krs.Spec.Replicas = &replica
	krs.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: d.Labels,
	}
	krs.Spec.Template = v1.PodTemplateSpec{}
	krs.Spec.Template.Labels = d.Labels
	krs.Spec.Template.Annotations = annotations
	krs.Spec.Template.Spec.Containers = append(krs.Spec.Template.Spec.Containers, container)
	krs.Spec.Template.Spec.Volumes = volumes
	//krs.Spec.Template.Spec.InitContainers = initContainers

	krs.Spec.Template.Labels["traffic"] = "yes"

	_, err = kclientset.ExtensionsV1beta1().ReplicaSets(d.Namespace).Create(&krs)
	if err != nil {
		return fmt.Errorf("Fail to create replicatSet: %s", err)
	}

	return nil
}

//ExposeService expose a service for the deployment
func (d *Deployment) ExposeService(kclientset kubernetes.Interface) error {

	svc := v1.Service{}
	svc.APIVersion = "v1"
	svc.Kind = "Service"
	svc.Name = d.Name
	svc.Namespace = d.Namespace
	svc.Spec.Selector = d.Labels
	svc.Spec.Selector["traffic"] = "yes"
	svc.Spec.Ports = []v1.ServicePort{v1.ServicePort{Port: 80}}
	svc.Spec.Type = "NodePort"

	_, err := kclientset.CoreV1().Services(d.Namespace).Create(&svc)
	if err != nil {
		return fmt.Errorf("Fail to create replicatSet: %s", err)
	}

	return nil
}
