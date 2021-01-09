package main

import (
	"fmt"
	"net/http"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"log"
	"path/filepath"

	resource "k8s.io/apimachinery/pkg/api/resource"
	"encoding/json"
	"errors"	
	"io/ioutil"
	
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"		

	"os"
)

const (
	tlsDir      = `/run/secrets/tls`
	tlsCertFile = `tls.crt`
	tlsKeyFile  = `tls.key`
)

var (
	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

// serveAdmitFunc is a wrapper around doServeAdmitFunc that adds error handling and logging.
func serveAdmitFunc(w http.ResponseWriter, r *http.Request) {
	log.Print("Handling webhook request ...")	
	
	var writeErr error
	if bytes, err := doServeAdmitFunc(w, r); err != nil {
			log.Printf("Error handling webhook request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_, writeErr = w.Write([]byte(err.Error()))
	} else {
			log.Print("Webhook request handled successfully")
			_, writeErr = w.Write(bytes)
	}

	if writeErr != nil {
			log.Printf("Could not write response: %v", writeErr)
	}

}


// doServeAdmitFunc parses the HTTP request for an admission controller webhook, and -- in case of a well-formed
// request -- delegates the admission control logic to the given admitFunc. The response body is then returned as raw
// bytes.
func doServeAdmitFunc(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	log.Print("In doServeAdmitFunc...")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not read request body: %v", err)
	}
	
	// parse request
	var admissionReviewReq v1beta1.AdmissionReview

	//admissionReviewReq, err := deserializeRequestBody(r)

	if _, _, err := universalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return nil, fmt.Errorf("could not deserialize request: %v", err)
    } else if admissionReviewReq.Request == nil {
        w.WriteHeader(http.StatusBadRequest)
        return nil, errors.New("malformed admission review: request is nil")
	}

	admissionReviewResponse := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
				UID: admissionReviewReq.Request.UID,
		},
	}

	// invoke validate method from main
	err = validateResources(admissionReviewReq.Request)
	
	if err != nil {
		// If the handler returned an error, incorporate the error message into the response and deny the object
		// creation.
		admissionReviewResponse.Response.Allowed = false
		admissionReviewResponse.Response.Result = &metav1.Status{
				Message: err.Error(),
		}
	} else {
		// Otherwise, return a positive response.	
		admissionReviewResponse.Response.Allowed = true		
    }	

	// Return the AdmissionReview with a response as JSON.
	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		return nil, fmt.Errorf("marshaling response: %v", err)
	}
	return bytes, nil
}

func main() {

	certPath := filepath.Join(tlsDir, tlsCertFile)
	keyPath := filepath.Join(tlsDir, tlsKeyFile)
	
	mux := http.NewServeMux()

	vh := http.HandlerFunc(serveAdmitFunc)

	mux.Handle("/validate", vh) 
		
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    ":8443",
		Handler: mux,
	}
	
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))

}

func validateResources(req *v1beta1.AdmissionRequest) (error) {
	log.Print("in validate resources method"+ req.UID)

	raw := req.Object.Raw
	//reqNamespace := req.Namespace
	deployment := v1.Deployment{}

	if _, _, err := universalDeserializer.Decode(raw, nil, &deployment); err != nil {
		return fmt.Errorf("could not deserialize deployment object: %v", err)
	}

	microserviceName := deployment.Spec.Template.Annotations["microserviceName"]

	microserviceSize := deployment.Spec.Template.Annotations["microserviceSize"]

	log.Print("microserviceName " + microserviceName + "  microserviceSize " + microserviceSize)

	// get the configured resources

	var container corev1.Container

	container = deployment.Spec.Template.Spec.Containers[0]

	log.Print("dealing with container:" + container.Name ) 
	
	//var requests corev1.ResourceList
	requests := container.Resources.Requests

	var cpu_requests resource.Quantity  
	cpu_requests =  requests["cpu"]

	var memory_requests resource.Quantity  
	memory_requests =  requests["memory"]

	fmt.Printf("memorySize = %v (%v)\n", memory_requests.Value(), memory_requests.Format)
	fmt.Printf("milliCores = %v (%v)\n", cpu_requests.MilliValue(), cpu_requests.Format)
	
	// create max allowed quantity for all sizes
	var ms_small_cpu_qty resource.Quantity
	ms_small_cpu_qty = resource.MustParse(os.Getenv("SMALL_MS_CPU_LIMIT"))

	var ms_small_mem_qty resource.Quantity
	ms_small_mem_qty = resource.MustParse(os.Getenv("SMALL_MS_MEM_LIMIT"))

	fmt.Printf("small memorySize = %v (%v)\n", ms_small_mem_qty.Value(), ms_small_mem_qty.Format)
	fmt.Printf("small milliCores = %v (%v)\n", ms_small_cpu_qty.MilliValue(), ms_small_cpu_qty.Format)

	var ms_med_cpu_qty resource.Quantity
	ms_med_cpu_qty = resource.MustParse(os.Getenv("MEDIUM_MS_CPU_LIMIT"))

	var ms_med_mem_qty resource.Quantity
	ms_med_mem_qty = resource.MustParse(os.Getenv("MEDIUM_MS_MEM_LIMIT"))

	fmt.Printf("medium memorySize = %v (%v)\n", ms_med_mem_qty.Value(), ms_med_mem_qty.Format)
	fmt.Printf("medium milliCores = %v (%v)\n", ms_med_cpu_qty.MilliValue(), ms_med_cpu_qty.Format)

	var ms_large_cpu_qty resource.Quantity
	ms_large_cpu_qty = resource.MustParse(os.Getenv("LARGE_MS_CPU_LIMIT"))

	var ms_large_mem_qty resource.Quantity
	ms_large_mem_qty = resource.MustParse(os.Getenv("LARGE_MS_MEM_LIMIT"))

	fmt.Printf("large memorySize = %v (%v)\n", ms_large_mem_qty.Value(), ms_large_mem_qty.Format)
	fmt.Printf("large milliCores = %v (%v)\n", ms_large_cpu_qty.MilliValue(), ms_large_cpu_qty.Format)

	// Cmp returns 0 if the quantity is equal to y, -1 if the quantity is less than y, or 1 if the
    // quantity is greater than y.

	fmt.Printf("comparison for small = %v (%v)\n:", cpu_requests.Cmp(ms_small_cpu_qty), memory_requests.Cmp(ms_small_mem_qty))
	fmt.Printf("comparison for med  = %v (%v)\n:", cpu_requests.Cmp(ms_med_cpu_qty), memory_requests.Cmp(ms_med_mem_qty))
	fmt.Printf("comparison for large  = %v (%v)\n:", cpu_requests.Cmp(ms_large_cpu_qty), memory_requests.Cmp(ms_large_mem_qty))


	if (microserviceSize == "S" && (cpu_requests.Cmp(ms_small_cpu_qty) > 0 || memory_requests.Cmp(ms_small_mem_qty) > 0)) {
		return fmt.Errorf("The microservice:" + microserviceName +" surpasses its cpu or memory limits set for its size. Allowed cpu is  %v (%v) and memory is  %v (%v)",ms_small_cpu_qty.MilliValue(), ms_small_cpu_qty.Format, ms_small_mem_qty.Value(), ms_small_mem_qty.Format)
	}
	if (microserviceSize == "M" && (cpu_requests.Cmp(ms_med_cpu_qty) > 0 || memory_requests.Cmp(ms_med_mem_qty) > 0)) {
		return fmt.Errorf("The microservice:" + microserviceName +" surpasses its cpu or memory limits set for its size.  Allowed cpu is  %v (%v) and memory is  %v (%v)",ms_med_cpu_qty.MilliValue(), ms_med_cpu_qty.Format, ms_med_mem_qty.Value(), ms_med_mem_qty.Format)
	}
	if (microserviceSize == "L" && (cpu_requests .Cmp(ms_large_cpu_qty) > 0 || memory_requests.Cmp(ms_large_mem_qty) > 0)) {
		return fmt.Errorf("The microservice:" + microserviceName +" surpasses its cpu or memory limits set for its size.  Allowed cpu is  %v (%v) and memory is  %v (%v)",ms_large_cpu_qty.MilliValue(),ms_large_cpu_qty.Format, ms_large_mem_qty.Value(), ms_large_mem_qty.Format)
	}

	return nil
}