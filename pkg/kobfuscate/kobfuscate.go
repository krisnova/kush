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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Runtime struct {
	client *kubernetes.Clientset
}

func NewRuntime() *Runtime {
	return &Runtime{}
}

func (r *Runtime) Hide() error {
	return nil
}

func (r *Runtime) Version() string {
	v, err := r.Client().ServerVersion()
	if err != nil {
		return "UNABLE TO DETECT VERSION"
	}
	return v.String()
}

// InCluster will ensure the client and see if we are running inside a cluster
func (r *Runtime) InCluster() (bool, error) {

	// Init client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return false, err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return false, err
	}
	r.client = client
	return true, nil
}

func (r *Runtime) Client() *kubernetes.Clientset {
	return r.client
}
