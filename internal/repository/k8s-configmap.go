package repository

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type K8sConfigMapRepository struct {
	client *kubernetes.Clientset
	ns     string
}

func NewK8sCMRepo(client *kubernetes.Clientset, ns string) *K8sConfigMapRepository {
	return &K8sConfigMapRepository{
		client: client,
		ns:     ns,
	}
}

func (r *K8sConfigMapRepository) ListRecipeConfigMaps(ctx context.Context) ([]corev1.ConfigMap, error) {
	list, err := r.client.CoreV1().ConfigMaps(r.ns).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=recipe-detection",
	})
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

func (r *K8sConfigMapRepository) CreateConfigMap(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return r.client.CoreV1().ConfigMaps(r.ns).Create(ctx, cm, metav1.CreateOptions{})
}

func (r *K8sConfigMapRepository) UpdateConfigMap(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return r.client.CoreV1().ConfigMaps(r.ns).Update(ctx, cm, metav1.UpdateOptions{})
}

func (r *K8sConfigMapRepository) DeleteConfigMap(ctx context.Context, name string) error {
	return r.client.CoreV1().ConfigMaps(r.ns).Delete(ctx, name, metav1.DeleteOptions{})
}

func (r *K8sConfigMapRepository) GetConfigMap(ctx context.Context, name string) (*corev1.ConfigMap, error) {
	return r.client.CoreV1().ConfigMaps(r.ns).Get(ctx, name, metav1.GetOptions{})
}
