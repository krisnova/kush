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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func HandleInject(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("%s%s", r.RemoteAddr, r.RequestURI)
	var body []byte
	var err error
	if r.Body != nil {
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			errstr := fmt.Sprintf("error reading body: %v", err)
			logrus.Errorf(errstr)
			http.Error(w, errstr, http.StatusBadRequest)
			return
		}
	}
	if len(body) == 0 {
		logrus.Errorf("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		logrus.Errorf("Content-Type: %s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	// Response
	var aResponse *admissionv1.AdmissionResponse

	// Review
	var aReview *admissionv1.AdmissionReview

	// Update aResponse with our mutation
	if _, _, err := deserializer.Decode(body, nil, aReview); err != nil {
		// Build an error response
		aResponse = &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		// Mutate aReview is now a Kubernetes object
		aResponse = mutate(aReview)
	}

	aReview = &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
	}

	// Prepare the response
	if aResponse != nil {
		aReview.Response = aResponse
		if aReview.Request != nil {
			aReview.Response.UID = aResponse.UID
		}
	}

	//aReview is what we write when we are done
	jsonData, err := json.Marshal(aReview)
	if err != nil {
		errstr := fmt.Sprintf("failed json.Marshal: %v", err)
		logrus.Errorf(errstr)
		http.Error(w, errstr, http.StatusUnsupportedMediaType)
		return
	}
	_, err = w.Write(jsonData)
	if err != nil {
		errstr := fmt.Sprintf("failed writing json: %v", err)
		logrus.Errorf(errstr)
		http.Error(w, errstr, http.StatusUnsupportedMediaType)
		return
	}
}

func mutate(in *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	var out *admissionv1.AdmissionResponse

	// TODO we need to mutate our requests here!

	return out
}
