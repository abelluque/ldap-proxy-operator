/*
Copyright 2025.
*/

package controller

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	proxyv1alpha1 "github.com/abelluque/proxy-ldap-operator/api/v1alpha1" // <-- IMPORTANTE: Reemplaza con la ruta de tu repositorio
)

// LdapProxyReconciler reconcilia un objeto LdapProxy
type LdapProxyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Marcadores RBAC: Permisos que necesita el Operador.
//+kubebuilder:rbac:groups=proxy.ar-consulting.redhat.com,resources=ldapproxies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=proxy.ar-consulting.redhat.com,resources=ldapproxies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=proxy.ar-consulting.redhat.com,resources=ldapproxies/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *LdapProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	ldapProxy := &proxyv1alpha1.LdapProxy{}

	err := r.Get(ctx, req.NamespacedName, ldapProxy)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Recurso LdapProxy no encontrado. Ignorando.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Falló al obtener LdapProxy")
		return ctrl.Result{}, err
	}

	// --- Reconciliar el Secret ---
	secret := r.secretForLdapProxy(ldapProxy)
	if err := ctrl.SetControllerReference(ldapProxy, secret, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, foundSecret)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creando un nuevo Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Create(ctx, secret)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// --- Reconciliar el Service ---
	service := r.serviceForLdapProxy(ldapProxy)
	if err := ctrl.SetControllerReference(ldapProxy, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creando un nuevo Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Create(ctx, service)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// --- Reconciliar el Deployment ---
	deployment := r.deploymentForLdapProxy(ldapProxy)
	if err := ctrl.SetControllerReference(ldapProxy, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creando un nuevo Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// --- Actualizar el Deployment si el número de réplicas cambia ---
	size := ldapProxy.Spec.Replicas
	if *foundDeployment.Spec.Replicas != size {
		foundDeployment.Spec.Replicas = &size
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			logger.Error(err, "Falló al actualizar el Deployment")
			return ctrl.Result{}, err
		}
	}

	// --- Actualizar el Status ---
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(ldapProxy.Namespace),
		client.MatchingLabels(labelsForLdapProxy(ldapProxy.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		logger.Error(err, "Falló al listar los pods")
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)
	if !reflect.DeepEqual(podNames, ldapProxy.Status.Nodes) {
		ldapProxy.Status.Nodes = podNames
		err := r.Status().Update(ctx, ldapProxy)
		if err != nil {
			logger.Error(err, "Falló al actualizar el status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// --- Funciones de ayuda con valores fijos ---

func (r *LdapProxyReconciler) deploymentForLdapProxy(m *proxyv1alpha1.LdapProxy) *appsv1.Deployment {
	ls := labelsForLdapProxy(m.Name)
	replicas := m.Spec.Replicas
	secretName := m.Name + "-secret"
	proxyImage := "quay.io/rhn-gps-aluque/proxy-ldap:1.0" // <-- VALOR FIJO
	containerPort := int32(1389)                          // <-- VALOR FIJO

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: m.Name, Namespace: m.Namespace, Labels: ls},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: ls},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: ls},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           proxyImage,
						Name:            "proxy",
						ImagePullPolicy: corev1.PullAlways,
						Ports: []corev1.ContainerPort{{
							ContainerPort: containerPort,
							Name:          "ldap-proxy",
							Protocol:      corev1.ProtocolTCP,
						}},
						Env: []corev1.EnvVar{
							{Name: "LISTEN_PORT", Value: "1389"},
						},
						EnvFrom: []corev1.EnvFromSource{{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
							}},
						},
						Resources: m.Spec.Resources,
					}},
				},
			},
		},
	}
	return dep
}

func (r *LdapProxyReconciler) serviceForLdapProxy(m *proxyv1alpha1.LdapProxy) *corev1.Service {
	ls := labelsForLdapProxy(m.Name)
	targetPort := int32(1389)
	ldapServicePort := int32(389)
	ldapsServicePort := int32(636)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: m.Name + "-svc", Namespace: m.Namespace, Labels: ls},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{
				{Name: "ldap", Port: ldapServicePort, TargetPort: intstr.FromInt(int(targetPort)), Protocol: corev1.ProtocolTCP},
				{Name: "ldaps", Port: ldapsServicePort, TargetPort: intstr.FromInt(int(targetPort)), Protocol: corev1.ProtocolTCP},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	return svc
}

func (r *LdapProxyReconciler) secretForLdapProxy(m *proxyv1alpha1.LdapProxy) *corev1.Secret {
	secretData := map[string]string{
		"LDAP_HOST":    m.Spec.LdapHost,
		"LDAP_PORT":    m.Spec.LdapPort,
		"LDAP_USE_TLS": m.Spec.LdapUseTLS,
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: m.Name + "-secret", Namespace: m.Namespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}
	return secret
}

func labelsForLdapProxy(name string) map[string]string {
	return map[string]string{"app": name}
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func (r *LdapProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&proxyv1alpha1.LdapProxy{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
