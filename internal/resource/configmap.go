// RabbitMQ Cluster Operator
//
// Copyright 2020 VMware, Inc. All Rights Reserved.
//
// This product is licensed to you under the Mozilla Public license, Version 2.0 (the "License").  You may not use this product except in compliance with the Mozilla Public License.
//
// This product may include a number of subcomponents with separate copyright notices and license terms. Your use of these subcomponents is subject to the terms and conditions of the subcomponent's license, as noted in the LICENSE file.
//

package resource

import (
	"fmt"
	"strings"

	rabbitmqv1beta1 "github.com/pivotal/rabbitmq-for-kubernetes/api/v1beta1"
	"github.com/pivotal/rabbitmq-for-kubernetes/internal/metadata"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	serverConfigMapName = "server-conf"
	defaultRabbitmqConf = `cluster_formation.peer_discovery_backend = rabbit_peer_discovery_k8s
cluster_formation.k8s.host = kubernetes.default
cluster_formation.k8s.address_type = hostname
cluster_formation.node_cleanup.interval = 30
cluster_formation.node_cleanup.only_log_warning = true
cluster_partition_handling = pause_minority
queue_master_locator = min-masters
`

	defaultTLSConf = `
ssl_options.certfile=/etc/rabbitmq-tls/tls.crt
ssl_options.keyfile=/etc/rabbitmq-tls/tls.key
listeners.ssl.default=5671
`

	defaultMutualTLSConf = `
ssl_options.verify = verify_peer
`
)

var RequiredPlugins = []string{
	"rabbitmq_peer_discovery_k8s", // required for clustering
	"rabbitmq_prometheus",         // enforce prometheus metrics
	"rabbitmq_management",
}

type ServerConfigMapBuilder struct {
	Instance *rabbitmqv1beta1.RabbitmqCluster
}

func (builder *RabbitmqResourceBuilder) ServerConfigMap() *ServerConfigMapBuilder {
	return &ServerConfigMapBuilder{
		Instance: builder.Instance,
	}
}

func (builder *ServerConfigMapBuilder) Update(object runtime.Object) error {
	configMap := object.(*corev1.ConfigMap)
	configMap.Labels = metadata.GetLabels(builder.Instance.Name, builder.Instance.Labels)
	configMap.Annotations = metadata.ReconcileAndFilterAnnotations(configMap.GetAnnotations(), builder.Instance.Annotations)

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	var rmqConfBuilder strings.Builder

	_, err := fmt.Fprintf(&rmqConfBuilder, "%scluster_name = %s\n", defaultRabbitmqConf, builder.Instance.Name)
	if err != nil {
		return err
	}

	if builder.Instance.TLSEnabled() {
		_, err := rmqConfBuilder.WriteString(defaultTLSConf)
		if err != nil {
			return err
		}
	}

	if builder.Instance.MutualTLSEnabled() {
		fmt.Fprintf(&rmqConfBuilder, "%s%s\n%s", "ssl_options.cacertfile=/etc/rabbitmq-tls/",
			builder.Instance.Spec.TLS.CaCertName,
			defaultMutualTLSConf)
	}

	// rabbitmq.conf takes the last provided value when multiple values of the same key are specified
	// do not need to deduplicate keys to allow overwrite
	if builder.Instance.Spec.Rabbitmq.AdditionalConfig != "" {
		_, err := rmqConfBuilder.WriteString(builder.Instance.Spec.Rabbitmq.AdditionalConfig)
		if err != nil {
			return err
		}
	}

	if builder.Instance.Spec.Rabbitmq.AdvancedConfig != "" {
		configMap.Data["advanced.config"] = builder.Instance.Spec.Rabbitmq.AdvancedConfig
	}
	configMap.Data["rabbitmq.conf"] = rmqConfBuilder.String()
	return nil
}

func (builder *ServerConfigMapBuilder) Build() (runtime.Object, error) {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.Instance.ChildResourceName(serverConfigMapName),
			Namespace: builder.Instance.Namespace,
		},
		Data: map[string]string{
			"enabled_plugins": "[" + strings.Join(AppendIfUnique(RequiredPlugins, builder.Instance.Spec.Rabbitmq.AdditionalPlugins), ",") + "].",
		},
	}, nil
}

func AppendIfUnique(a []string, b []rabbitmqv1beta1.Plugin) []string {
	data := make([]string, len(b))
	for i := range data {
		data[i] = string(b[i])
	}

	check := make(map[string]bool)
	list := append(a, data...)
	set := make([]string, 0)
	for _, s := range list {
		if _, value := check[s]; !value {
			check[s] = true
			set = append(set, s)
		}
	}
	return set
}
