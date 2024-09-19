package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var managedByLabels = parseManagedByLabels()

func parseManagedByLabels() map[string]string {
	labels := make(map[string]string)
	envValue := os.Getenv("MANAGED_BY_LABEL")

	if envValue == "" {
		labels["authorino.kuadrant.io/managed-by"] = "authorino"
		return labels
	}

	labelPairs := strings.Split(envValue, ",")
	for _, pair := range labelPairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			labels[key] = value
		}
	}

	return labels
}

type Secret struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
	Data      map[string]string `json:"data"`
}

func createAPIKeySecret(clientset *kubernetes.Clientset, secret Secret) error {
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}

	for k, v := range managedByLabels {
		secret.Labels[k] = v
	}

	stringData := make(map[string]string)
	for k, v := range secret.Data {
		stringData[k] = v
	}

	secretObject := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Labels:    secret.Labels,
		},
		StringData: stringData,
		Type:       v1.SecretTypeOpaque,
	}

	_, err := clientset.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secretObject, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret: %v", err)
	}
	return nil
}

func updateAPIKeySecret(clientset *kubernetes.Clientset, secret Secret) error {
	existingSecret, err := clientset.CoreV1().Secrets(secret.Namespace).Get(context.TODO(), secret.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret for update: %v", err)
	}

	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}

	for k, v := range managedByLabels {
		secret.Labels[k] = v
	}

	stringData := make(map[string]string)
	for k, v := range secret.Data {
		stringData[k] = v
	}

	existingSecret.Labels = secret.Labels
	existingSecret.StringData = stringData

	_, err = clientset.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret: %v", err)
	}

	return nil
}

func deleteAPIKeySecret(clientset *kubernetes.Clientset, namespace string, secretName string) error {
	err := clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), secretName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret: %v", err)
	}
	return nil
}

func getAPIKeySecrets(clientset *kubernetes.Clientset, namespace string) ([]Secret, error) {
	labelSelectorParts := []string{}
	for k, v := range managedByLabels {
		labelSelectorParts = append(labelSelectorParts, fmt.Sprintf("%s=%s", k, v))
	}
	labelSelector := strings.Join(labelSelectorParts, ",")

	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %v", err)
	}

	var secretList []Secret

	for _, secret := range secrets.Items {
		data := make(map[string]string)
		for k, v := range secret.Data {
			data[k] = string(v)
		}

		secretList = append(secretList, Secret{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Labels:    secret.Labels,
			Data:      data,
		})
	}

	return secretList, nil
}

func handleAPIKeySecrets(clientset *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			namespace := r.URL.Query().Get("namespace")
			if namespace == "" {
				http.Error(w, "Namespace is required", http.StatusBadRequest)
				return
			}

			secrets, err := getAPIKeySecrets(clientset, namespace)
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

			err = createAPIKeySecret(clientset, secretReq)
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

			err = updateAPIKeySecret(clientset, secretReq)
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

			err := deleteAPIKeySecret(clientset, namespace, secretName)
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
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to load in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/secrets", handleAPIKeySecrets(clientset))

	port := ":8080"
	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
