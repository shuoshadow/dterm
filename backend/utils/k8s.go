package utils

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"
)

type K8sClient struct {
	name      string
	clientset *kubernetes.Clientset
}

type K8sConfig struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Kubeconfig string `json:"kubeconfig"`
}

type K8sConfigList []*K8sConfig

func NewK8sClient(c *K8sConfig) *K8sClient {
	config := &rest.Config{}
	err := errors.New("")
	switch c.Type {
	case "out-of-cluster":
		kubeconfig := "/opt/dterm/.kube/" + c.Kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	default:
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return &K8sClient{
		name:      c.Name,
		clientset: clientset,
	}
}

type PodEvent struct {
	Type  string
	Event *apiv1.Pod
}

func (k *K8sClient) WatchPod(ns string) <-chan *PodEvent {
	//init es client
	// esClient := NewESClient()
	// watch pods
	ch := make(chan *PodEvent, 1024)
	go func(ns string, ch chan<- *PodEvent) {

	loop:
		w, err := k.clientset.CoreV1().Pods(ns).Watch(metav1.ListOptions{})
		if err != nil {
			log.Error(err)
			// need retry here
			time.Sleep(30 * time.Second)
			goto loop
		}
		for {
			select {
			case e, ok := <-w.ResultChan():
				if !ok {
					log.Info("watch channel closed, restart the loop.")
					// reindex 'pods' doesn't work when there're more than one k8s cluster
					// err = esClient.ReIndex("pods")
					// if err != nil {
					// 	log.Error(fmt.Sprintf("failed to reindex 'pods', %v", err))
					// }
					goto loop
				}
				podObj := e.Object.(*apiv1.Pod) //need error handle here
				podRef := podObj.GetOwnerReferences()
				podRefNone := metav1.OwnerReference{
					Kind: "None",
					Name: k.name + ":" + podObj.ObjectMeta.Namespace + ":" + "pods",
				}
				if len(podRef) == 0 {
					podObj.SetOwnerReferences([]metav1.OwnerReference{podRefNone})
				} else {
					if len(podRef) > 1 {
						log.Warn(fmt.Sprintf("there's more than one reference, %v", podRef))
					}
					podRef[0].Name = k.name + ":" + podObj.ObjectMeta.Namespace + ":" + podRef[0].Name
					podObj.SetOwnerReferences(podRef)
				}

				p := &PodEvent{
					Type:  string(e.Type),
					Event: podObj,
				}
				ch <- p
			}
		}
	}(ns, ch)

	return ch
}
