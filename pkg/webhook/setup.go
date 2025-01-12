// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package webhook

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"


	clustermanager "github.com/open-cluster-management/api/operator/v1"
	admissionregistration "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	operatorsv1 "github.com/open-cluster-management/multiclusterhub-operator/pkg/apis/operator/v1"
	"github.com/open-cluster-management/multiclusterhub-operator/pkg/utils"
)

var log = logf.Log.WithName("multiclusterhub_webhook")

const (
	resourceName          = "multiclusterhubs"
	clusterManagerName    = "clustermanagers"
	operatorName          = "multiclusterhub-operator"
	validatingWebhookName = "multiclusterhub.validating-webhook.open-cluster-management.io"
	validatingCfgName     = "multiclusterhub-operator-validating-webhook"
)

func Setup(mgr manager.Manager) error {
	certDir := filepath.Join("/tmp", "webhookcert")

	
	ns, err := utils.FindNamespace()
	if err != nil {
		return err
	}

	done := make(chan string)
	go createOrUpdateWebhookService(mgr.GetClient(), ns)


	go getSecret(mgr.GetClient(), ns, certDir, done)

	go func() {
		for {
			select {
			case x, ok := <-done:
				if ok && x == "success" {
					log.Info("Secret has been successfully read. Moving to create service.")
					finalizeWebhook(mgr, ns, certDir)
					return
				} else {
					log.Info("Channel returned with no secret value")
					return
				}
			default:
				log.Info("Still waiting for secret value.")
			}
			time.Sleep(5 * time.Second)
		}
	}()

	return nil
}

func finalizeWebhook(mgr manager.Manager, ns, certDir string) error {
	hookServer := &webhook.Server{
		Port:    8443,
		CertDir: certDir,
	}

	log.Info("Add the webhook server.")
	if err := mgr.Add(hookServer); err != nil {
		return err
	}

	log.Info("Registering webhooks to the webhook server.")
	validatingPath := "/validate-v1-multiclusterhub"
	hookServer.Register(validatingPath, &webhook.Admission{Handler: &multiClusterHubValidator{}})

	go createOrUpdateValiatingWebhook(mgr.GetClient(), ns, validatingPath, certDir)
	return nil
}

func createOrUpdateWebhookService(c client.Client, namespace string) {
	log.Info("service function called")
	service := &corev1.Service{}
	key := types.NamespacedName{Name: utils.WebhookServiceName, Namespace: namespace}
	for {
		if err := c.Get(context.TODO(), key, service); err != nil {
			if errors.IsNotFound(err) {

				service := newWebhookService(namespace)
				setOwnerReferences(c, namespace, service)
				if err := c.Create(context.TODO(), service); err != nil {
					log.Error(err, fmt.Sprintf("Failed to create %s/%s service", namespace, utils.WebhookServiceName))
					return
				}
				log.Info(fmt.Sprintf("Create %s/%s service", namespace, utils.WebhookServiceName))

				return
			}
			switch err.(type) {
			case *cache.ErrCacheNotStarted:
				time.Sleep(time.Second)
				continue
			default:
				log.Error(err, fmt.Sprintf("Failed to get %s/%s service", namespace, utils.WebhookServiceName))
				return
			}
		}		
		metav1.SetMetaDataAnnotation(&service.ObjectMeta, "service.beta.openshift.io/serving-cert-secret-name", "multiclusterhub-operator-webhook") 
		if err := c.Update(context.TODO(), service); err != nil {
			log.Error(err, fmt.Sprintf("Failed to update service %s", utils.WebhookServiceName))
			return
		}
		return
	}
}

func createOrUpdateValiatingWebhook(c client.Client, namespace, path string, certDir string) {

	validator := &admissionregistration.ValidatingWebhookConfiguration{}
	key := types.NamespacedName{Name: validatingCfgName}
	for {
		if err := c.Get(context.TODO(), key, validator); err != nil {
			if errors.IsNotFound(err) {
				cfg := newValidatingWebhookCfg(namespace, path)
				setOwnerReferences(c, namespace, cfg)
				if err := c.Create(context.TODO(), cfg); err != nil {
					log.Error(err, fmt.Sprintf("Failed to create validating webhook %s", validatingCfgName))
					return
				}
				log.Info(fmt.Sprintf("Create validating webhook %s", validatingCfgName))
				return
			}
			switch err.(type) {
			case *cache.ErrCacheNotStarted:
				time.Sleep(time.Second)
				continue
			default:
				log.Error(err, fmt.Sprintf("Failed to get validating webhook %s", validatingCfgName))
				return
			}
		}

		
		metav1.SetMetaDataAnnotation(&validator.ObjectMeta, "service.beta.openshift.io/inject-cabundle", "true")
		if err := c.Update(context.TODO(), validator); err != nil {
			log.Error(err, fmt.Sprintf("Failed to update validating webhook %s", validatingCfgName))
			return
		}
		log.Info(fmt.Sprintf("Update validating webhook %s", validatingCfgName))
		return
	}
}

func setOwnerReferences(c client.Client, namespace string, obj metav1.Object) {
	key := types.NamespacedName{Name: operatorName, Namespace: namespace}
	owner := &appsv1.Deployment{}
	if err := c.Get(context.TODO(), key, owner); err != nil {
		log.Error(err, fmt.Sprintf("Failed to set owner references for %s", obj.GetName()))
		return
	}

	obj.SetOwnerReferences([]metav1.OwnerReference{
		*metav1.NewControllerRef(owner, owner.GetObjectKind().GroupVersionKind())})
}

func getSecret(c client.Client, namespace string, certDir string, done chan<- string) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Name: "multiclusterhub-operator-webhook", Namespace: namespace}

	for {
		if err := c.Get(context.TODO(), key, secret); err != nil {

			switch err.(type) {
			case *cache.ErrCacheNotStarted:
				time.Sleep(time.Second)
				continue
			default:
				time.Sleep(time.Second)
				log.Error(err, fmt.Sprintf("Fails to return secret"))
				continue
			}
		}
		log.Info(fmt.Sprintf("Successfully returned secret"))

		if err := os.MkdirAll(certDir, os.ModePerm); err != nil {
			log.Error(err, fmt.Sprintf("trouble creating directory"))
			return
		}
		if err := ioutil.WriteFile(filepath.Join(certDir, "tls.crt"), secret.Data["tls.crt"], os.FileMode(0644)); err != nil {
			log.Error(err, fmt.Sprintf("trouble writing crt"))
			return
		}
		if err := ioutil.WriteFile(filepath.Join(certDir, "tls.key"), secret.Data["tls.key"], os.FileMode(0644)); err != nil {
			log.Error(err, fmt.Sprintf("trouble writing key"))
			return
		}
		done <- "success"
		return
	}
}

func newServer(certDir string) *webhook.Server {

	for {
		if _, err := os.Stat(filepath.Join(certDir, "tls.crt")); err != nil {
			time.Sleep(time.Second)
			log.Error(err, fmt.Sprintf("not finding file"))
			continue
		}
		log.Info(fmt.Sprintf("found file"))
		return &webhook.Server{
			Port:    8443,
			CertDir: certDir,
		}
	}
}

func newWebhookService(namespace string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        utils.WebhookServiceName,
			Namespace:   namespace,
			Annotations: map[string]string{"service.beta.openshift.io/serving-cert-secret-name": "multiclusterhub-operator-webhook"},
		},
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{{Port: 443, TargetPort: intstr.FromInt(8443)}},
			Selector: map[string]string{"name": operatorName},
		},
	}
}

func newValidatingWebhookCfg(namespace, path string) *admissionregistration.ValidatingWebhookConfiguration {
	sideEffect := admissionregistration.SideEffectClassNone

	return &admissionregistration.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:        validatingCfgName,
			Annotations: map[string]string{"service.beta.openshift.io/inject-cabundle": "true"},
		},
		Webhooks: []admissionregistration.ValidatingWebhook{{
			AdmissionReviewVersions: []string{
				"v1beta1",
			},
			ClientConfig: admissionregistration.WebhookClientConfig{
				Service: &admissionregistration.ServiceReference{
					Name:      utils.WebhookServiceName,
					Namespace: namespace,
					Path:      &path,
				},
			},
			Name: validatingWebhookName,
			Rules: []admissionregistration.RuleWithOperations{{
				Rule: admissionregistration.Rule{
					APIGroups:   []string{operatorsv1.SchemeGroupVersion.Group},
					APIVersions: []string{operatorsv1.SchemeGroupVersion.Version},
					Resources:   []string{resourceName},
				},
				Operations: []admissionregistration.OperationType{
					admissionregistration.Create,
					admissionregistration.Update,
					admissionregistration.Delete,
				},
			}, {
				Rule: admissionregistration.Rule{
					APIGroups:   []string{clustermanager.SchemeGroupVersion.Group},
					APIVersions: []string{clustermanager.SchemeGroupVersion.Version},
					Resources:   []string{clusterManagerName},
				},
				Operations: []admissionregistration.OperationType{
					admissionregistration.Delete,
				},
			}},
			SideEffects: &sideEffect,
		}},
	}
}
