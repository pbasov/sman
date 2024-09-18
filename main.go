package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Secret represents the structure for both creating and retrieving secrets
type Secret struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Data      map[string]string `json:"data"`
}

// createSecret is a reusable function to create a Kubernetes secret
func createSecret(clientset *kubernetes.Clientset, namespace string, secretName string, secretData map[string][]byte) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: secretData,
	}

	_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret: %v", err)
	}
	return nil
}

// getSecrets returns the list of secrets from a specific namespace
func getSecrets(clientset *kubernetes.Clientset, namespace string) ([]Secret, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %v", err)
	}

	var secretList []Secret

	// Extract the secrets data and return them in a simplified format
	for _, secret := range secrets.Items {
		secretData := make(map[string]string)
		for key, value := range secret.Data {
			secretData[key] = string(value) // Convert from byte array to string
		}

		secretList = append(secretList, Secret{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Data:      secretData,
		})
	}

	return secretList, nil
}

// handleSecrets handles both GET and POST requests to create or retrieve secrets
func handleSecrets(clientset *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// Handle GET request to retrieve secrets
			namespace := r.URL.Query().Get("namespace")
			if namespace == "" {
				http.Error(w, "Namespace is required", http.StatusBadRequest)
				return
			}

			secrets, err := getSecrets(clientset, namespace)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error fetching secrets: %v", err), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(secrets); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}

		case http.MethodPost:
			// Handle POST request to create a new secret
			var secretReq Secret

			// Parse JSON request
			err := json.NewDecoder(r.Body).Decode(&secretReq)
			if err != nil {
				http.Error(w, "Invalid request payload", http.StatusBadRequest)
				return
			}

			// Convert secret data from string to byte array
			secretData := make(map[string][]byte)
			for key, value := range secretReq.Data {
				secretData[key] = []byte(value)
			}

			// Call the reusable function to create the secret
			err = createSecret(clientset, secretReq.Namespace, secretReq.Name, secretData)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error creating secret: %v", err), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, "Secret %s created successfully in namespace %s\n", secretReq.Name, secretReq.Namespace)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func main() {
	// In-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to load in-cluster config: %v", err)
	}

	// Create Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Serve static files from the "frontend" directory
	fs := http.FileServer(http.Dir("./frontend"))
	http.Handle("/", fs)

	// Serve the API endpoint for both creating and retrieving secrets
	http.HandleFunc("/secrets", handleSecrets(clientset))

	// Start the HTTP server
	port := ":8080"
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
