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

type Secret struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Data      map[string]string `json:"data"`
}

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

func updateSecret(clientset *kubernetes.Clientset, namespace string, secretName string, secretData map[string][]byte) error {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret for update: %v", err)
	}

	// Update the secret data
	secret.Data = secretData

	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret: %v", err)
	}

	return nil
}

func deleteSecret(clientset *kubernetes.Clientset, namespace string, secretName string) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), secretName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret: %v", err)
	}
	return nil
}

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

// handleSecrets handles GET, POST, PUT, and DELETE requests for secrets
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
			var secretReq Secret
			err := json.NewDecoder(r.Body).Decode(&secretReq)
			if err != nil {
				http.Error(w, "Invalid request payload", http.StatusBadRequest)
				return
			}

			secretData := make(map[string][]byte)
			for key, value := range secretReq.Data {
				secretData[key] = []byte(value)
			}

			err = createSecret(clientset, secretReq.Namespace, secretReq.Name, secretData)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error creating secret: %v", err), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, "Secret %s created successfully in namespace %s\n", secretReq.Name, secretReq.Namespace)

		case http.MethodPut:
			var secretReq Secret

			err := json.NewDecoder(r.Body).Decode(&secretReq)
			if err != nil {
				http.Error(w, "Invalid request payload", http.StatusBadRequest)
				return
			}

			secretData := make(map[string][]byte)
			for key, value := range secretReq.Data {
				secretData[key] = []byte(value)
			}

			err = updateSecret(clientset, secretReq.Namespace, secretReq.Name, secretData)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error updating secret: %v", err), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Secret %s updated successfully in namespace %s\n", secretReq.Name, secretReq.Namespace)

		case http.MethodDelete:
			namespace := r.URL.Query().Get("namespace")
			secretName := r.URL.Query().Get("name")
			if namespace == "" || secretName == "" {
				http.Error(w, "Namespace and secret name are required", http.StatusBadRequest)
				return
			}

			err := deleteSecret(clientset, namespace, secretName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error deleting secret: %v", err), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Secret %s deleted successfully from namespace %s\n", secretName, namespace)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func main() {
	// in-cluster client using service account
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to load in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// serve frontend
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// api endpoint
	http.HandleFunc("/secrets", handleSecrets(clientset))

	port := ":8080"
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
