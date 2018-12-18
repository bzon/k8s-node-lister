package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// "k8s.io/api/core/v1"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type node struct {
	IP     string `json:"ip"`
	AZ     string `json:"az"`
	Region string `json:"region"`
}

func main() {
	var kubeconfig *string
	useInCluster := flag.Bool("in-cluster", false, "Set to true to use in cluster kubernetes configuration")
	findLabelsWith := flag.String("with-label", "node-role.kubernetes.io/node", "Get nodes with label 'key'")
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	var config = &rest.Config{}
	var err error
	if *useInCluster {
		log.Info("Using in cluster config to connect to the cluster.")
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		log.Infof("Using kubeconfig: %q to connect to the cluster", *kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Get the nodes
	matchednodes := make(map[string][]node)
	go func() {
		for {
			log.Debug("Fetching new nodes from the Kubernetes cluster")
			nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
			if err != nil {
				log.Error(err.Error())
			}
			// empty the matchednodes map
			matchednodes = make(map[string][]node)
			for _, n := range nodes.Items {
				for nodelabel := range n.Labels {
					var matched bool
					if strings.Contains(nodelabel, *findLabelsWith) {
						matched = true
					}
					if matched {
						matchednode := node{
							IP:     n.Labels["kubernetes.io/hostname"],
							AZ:     n.Labels["failure-domain.beta.kubernetes.io/zone"],
							Region: n.Labels["failure-domain.beta.kubernetes.io/region"],
						}
						matchednodes[nodelabel] = append(matchednodes[nodelabel], matchednode)
					}
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()

	// Create the http server
	mux := http.NewServeMux()
	mux.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		bs, err := json.Marshal(matchednodes)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(bs)
	})
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
