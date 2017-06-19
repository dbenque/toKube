package deployer

import (
	"encoding/json"
	"flag"
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	kapi "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var (
	cpuLimit      string
	cpuRequest    string
	memoryLimit   string
	memoryRequest string
	namespace     string
	baseImage     string
	replicas      int
	labels        string
	configMaps    string
)

func init() {
	flag.StringVar(&cpuLimit, "cpu-limit", "100m", "Max CPU in milicores")
	flag.StringVar(&cpuRequest, "cpu-request", "100m", "Min CPU in milicores")
	flag.StringVar(&memoryLimit, "memory-limit", "64M", "Max memory in MB")
	flag.StringVar(&memoryRequest, "memory-request", "64M", "Min memory in MB")
	flag.StringVar(&namespace, "namespace", "default", "The Kubernetes namespace.")
	flag.IntVar(&replicas, "replicas", 1, "Number of replicas")
	flag.StringVar(&baseImage, "base-image", "alpine:3.4", "Base image to run the container")
	flag.StringVar(&labels, "labels", "{}", "map of labels (json serialization of map)")
	flag.StringVar(&configMaps, "configMaps", "[]", "list of configMap name to mount in volumes (json serialization of map)")
}

//Deployment contains configuration for the deployment
type Deployment struct {
	Annotations     map[string]string
	Args            []string
	Env             map[string]string
	BinaryURL       string
	Resource        kapi.ResourceRequirements
	Name            string
	Namespace       string
	Replicas        int
	Labels          map[string]string // Labels on rs,service and pod
	PodLabels       map[string]string // Extension of Pod Labels.
	ConfigMapVolume []string
}

//NewDeploymentFromArgs prepare a deployment based on the command line parameters
func NewDeploymentFromArgs(name string) *Deployment {
	d := &Deployment{
		Resource: kapi.ResourceRequirements{
			Limits: kapi.ResourceList{
				kapi.ResourceCPU:    resource.MustParse(cpuLimit),
				kapi.ResourceMemory: resource.MustParse(memoryLimit),
			},
			Requests: kapi.ResourceList{
				kapi.ResourceCPU:    resource.MustParse(cpuRequest),
				kapi.ResourceMemory: resource.MustParse(memoryRequest),
			},
		},
		Replicas:        replicas,
		Namespace:       namespace,
		Annotations:     map[string]string{},
		Labels:          map[string]string{},
		PodLabels:       map[string]string{},
		Env:             map[string]string{},
		Args:            []string{},
		ConfigMapVolume: []string{},
		Name:            name,
	}

	lbs := map[string]string{}
	if err := json.Unmarshal([]byte(labels), &lbs); err == nil {
		for k, v := range lbs {
			d.PodLabels[k] = v
		}
	} else {
		//TODO : log error
	}

	if err := json.Unmarshal([]byte(configMaps), &d.ConfigMapVolume); err != nil {
		//TODO : log error
	}

	return d
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

	// configMap volumes
	for _, cmv := range d.ConfigMapVolume {
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "config-" + cmv,
			MountPath: "/cfg/" + cmv,
		})
		volumes = append(volumes, v1.Volume{
			Name: "config-" + cmv,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: cmv},
				},
			},
		})
	}

	container := v1.Container{}
	container.Args = d.Args
	container.Command = []string{filepath.Join("/opt/bin", d.Name)}
	container.Image = baseImage
	container.Name = d.Name
	container.VolumeMounts = volumeMounts
	container.Ports = []v1.ContainerPort{v1.ContainerPort{ContainerPort: 9102}} // for prometheus

	// resourceLimits := make(v1.ResourceList)
	// if d.cpuLimit != "" {
	// 	resourceLimits[v1.ResourceCPU] = resource.MustParse(d.cpuLimit)
	// }
	// if d.memoryLimit != "" {
	// 	resourceLimits[v1.ResourceMemory] = resource.MustParse(d.memoryLimit)
	// }

	// resourceRequests := make(v1.ResourceList)
	// if d.cpuRequest != "" {
	// 	resourceRequests[v1.ResourceCPU] = resource.MustParse(d.cpuRequest)
	// }
	// if d.memoryRequest != "" {
	// 	resourceRequests[v1.ResourceMemory] = resource.MustParse(d.memoryRequest)
	// }

	// if len(resourceLimits) > 0 {
	// 	container.Resources.Limits = resourceLimits
	// }
	// if len(resourceRequests) > 0 {
	// 	container.Resources.Requests = resourceRequests
	// }

	container.Resources = d.Resource

	if len(d.Env) > 0 {
		env := make([]v1.EnvVar, 0)
		for name, value := range d.Env {
			env = append(env, v1.EnvVar{Name: name, Value: value})
		}
		container.Env = env
	}

	if d.Annotations == nil {
		d.Annotations = map[string]string{}
	}
	annotations := d.Annotations

	binaryPath := filepath.Join("/opt/bin", d.Name)
	initContainers := []v1.Container{
		v1.Container{
			Name:            "install",
			Image:           "alpine:3.4",
			ImagePullPolicy: "IfNotPresent",
			Command:         []string{"wget", "-O", binaryPath, d.BinaryURL},
			VolumeMounts: []v1.VolumeMount{
				v1.VolumeMount{
					Name:      "bin",
					MountPath: "/opt/bin",
				},
			},
		},
		v1.Container{
			Name:            "configure",
			Image:           "alpine:3.4",
			ImagePullPolicy: "IfNotPresent",
			Command:         []string{"chmod", "+x", binaryPath},
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
	d.Labels["visualize"] = "true"

	var replica int32
	replica = int32(d.Replicas)

	krs := v1beta1.ReplicaSet{}
	krs.APIVersion = "extensions/v1beta1"
	krs.Kind = "ReplicaSet"
	krs.Name = d.Name
	krs.Namespace = d.Namespace
	krs.Spec = v1beta1.ReplicaSetSpec{}
	krs.Spec.Replicas = &replica
	selector := map[string]string{}
	for k, v := range d.Labels {
		switch k {
		case "name", "visualize":
		// ignore these ones
		default:
			selector[k] = v
		}
	}

	krs.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: selector,
	}
	krs.Labels = d.Labels
	krs.Spec.Template = v1.PodTemplateSpec{}

	krs.Spec.Template.Annotations = annotations
	krs.Spec.Template.Spec.Containers = append(krs.Spec.Template.Spec.Containers, container)
	krs.Spec.Template.Spec.Volumes = volumes
	//krs.Spec.Template.Spec.InitContainers = initContainers
	podTemplateLabels := map[string]string{}
	for k, v := range selector {
		podTemplateLabels[k] = v
	}
	podTemplateLabels["visualize"] = "true" // Label extension compare to selector
	podTemplateLabels["traffic"] = "yes"

	for k, v := range d.PodLabels {
		podTemplateLabels[k] = v
	}

	krs.Spec.Template.Labels = podTemplateLabels

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
	svc.Labels = map[string]string{"visualize": "true", "run": d.Name}
	svc.Spec.Selector = d.Labels
	svc.Spec.Selector["traffic"] = "yes"
	svc.Spec.Ports = []v1.ServicePort{v1.ServicePort{Port: 80, Name: "http"}}
	svc.Spec.Type = "NodePort"

	_, err := kclientset.CoreV1().Services(d.Namespace).Create(&svc)
	if err != nil {
		return fmt.Errorf("Fail to create service: %s", err)
	}

	return nil
}
