package kubernetes

import (
	"context"
	"flag"
	"path/filepath"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const LABEL_OPERATING_SYSTEM = "kubernetes.io/os"
const LABEL_OPERATING_SYSTEM_LINUX = "Linux"
const LABEL_OPERATING_SYSTEM_WINDOWS = "Windows"

func CreateClientSet() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func GetNodes(c *kubernetes.Clientset) ([]v1.Node, error) {
	nodeList, err := c.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return nodeList.Items, nil
}
