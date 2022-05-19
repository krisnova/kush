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
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/util/homedir"
)

const (
	NSCatUtil string = "nscat"
)

var (
	// KubeconfigLocations will be the list
	// of kubeconfig locations on the host to
	// consider!
	//
	// Note: Order is important as the program
	// will return the first found kubeconfig!
	KubeconfigLocations = []string{
		"/etc/kubernetes/admin.conf",
		"/root/.kube/config",
		"~/.kube/config",
	}

	NSCatHostMountFlags = []string{
		"-t", // Target
		"1",  // Pid 1
		"-m", // Mount Namespace
	}
)

func HostKubeConfig() (string, error) {
	for _, kubeconfig := range KubeconfigLocations {
		kubeconfig = strings.ReplaceAll(kubeconfig, "~", homedir.HomeDir())
		logrus.Debugf("Trying: %s", kubeconfig)
		out := NSCat(kubeconfig)
		if out != "" {
			return out, nil
		}
	}
	// Unable to find the host kubeconfig
	return "", fmt.Errorf("unable to find kubeconfig in host mount namepsaces")
}

func NSCat(src string) string {
	args := append(NSCatHostMountFlags, src)
	logrus.Debugf("Executing: %s %s", NSCatUtil, strings.Join(args, " "))
	cmd := exec.Command(NSCatUtil, args...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Run()
	out := stdout.String()
	err := stderr.String()
	if err != "" {
		logrus.Errorf("%s", err)
	}
	return out
}
