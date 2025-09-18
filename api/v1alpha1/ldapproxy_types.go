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

	// Image es la imagen de contenedor a usar para el proxy ldap.
	// Contiene la URL completa de tu imagen en Quay.io.
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// --- Configuración del Secret ---
	// LdapHost es la dirección del servidor LDAP al que el proxy se conectará.
	LdapHost string `json:"ldapHost"`
	// LdapPort es el puerto del servidor LDAP.
	LdapPort string `json:"ldapPort"`
	// LdapUseTLS indica si se debe usar TLS para conectar al servidor LDAP.
	LdapUseTLS string `json:"ldapUseTLS"`

	// --- Configuración del Service ---
	// TargetPort es el puerto en el que escucha el contenedor del proxy.
	// +kubebuilder:default=1389
	TargetPort int32 `json:"targetPort"`
	// LdapServicePort es el puerto que expondrá el Service para el tráfico LDAP.
	// +kubebuilder:default=389
	LdapServicePort int32 `json:"ldapServicePort"`
	// LdapsServicePort es el puerto que expondrá el Service para el tráfico LDAPS.
	// +kubebuilder:default=636
	LdapsServicePort int32 `json:"ldapsServicePort"`

	// Resources define los requests y limits de CPU/Memoria para el contenedor.
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// LdapProxyStatus define el estado observado de LdapProxy
type LdapProxyStatus struct {
	// PodNames son los nombres de los pods que están corriendo el proxy.
	Nodes []string `json:"nodes"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
type LdapProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LdapProxySpec   `json:"spec,omitempty"`
	Status LdapProxyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
type LdapProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LdapProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LdapProxy{}, &LdapProxyList{})
}
