/*
Copyright 2025.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LdapProxySpec define el estado deseado de LdapProxy
type LdapProxySpec struct {
	// Replicas es el número de pods que se ejecutarán para el proxy.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// --- Configuración del Secret ---
	// LdapHost es la dirección del servidor LDAP al que el proxy se conectará.
	// +kubebuilder:validation:Required
	LdapHost string `json:"ldapHost"`

	// LdapPort es el puerto del servidor LDAP.
	// +kubebuilder:validation:Required
	LdapPort string `json:"ldapPort"`

	// LdapUseTLS indica si se debe usar TLS para conectar al servidor LDAP.
	// +kubebuilder:validation:Required
	LdapUseTLS string `json:"ldapUseTLS"`

	// Resources define los requests y limits de CPU/Memoria para el contenedor.
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// LdapProxyStatus define el estado observado de LdapProxy
type LdapProxyStatus struct {
	// Nodes son los nombres de los pods que están corriendo el proxy.
	Nodes []string `json:"nodes"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LdapProxy es el Schema para la API de ldapproxies
type LdapProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LdapProxySpec   `json:"spec,omitempty"`
	Status LdapProxyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LdapProxyList contiene una lista de LdapProxy
type LdapProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LdapProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LdapProxy{}, &LdapProxyList{})
}
