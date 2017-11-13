#!/bin/sh

# Make sure to have mounted /host/opt/cni/bin /host/etc/cni/net.d
# Make sure CNI_GENIE_NETWORK_CONFIG config map env is set.

# Ensure all variables are defined.
set -u

# Default CNI networks directory and security certificates location
HOST_CNI_NET_DIR=${CNI_NET_DIR:-/etc/cni/net.d}
HOST_SECRETS_DIR=${HOST_CNI_NET_DIR}/genie-tls

# Directory where we expect that TLS assets will be mounted into
# the genie/cni container.
SECRETS_MOUNT_DIR=${TLS_ASSETS_DIR:-/genie-secrets}

# Clean up any existing binaries / config / assets.
rm -f /host/opt/cni/bin/genie
rm -f /host/etc/cni/net.d/genie-tls/*

# Copy over any TLS assets from the SECRETS_MOUNT_DIR to the host.
if [ -e "${SECRETS_MOUNT_DIR}" ];
then
        echo "Installing any TLS assets from ${SECRETS_MOUNT_DIR}"
        mkdir -p /host/etc/cni/net.d/genie-tls
        cp ${SECRETS_MOUNT_DIR}/* /host/etc/cni/net.d/genie-tls/
fi

# If the TLS assets actually exist, update the variables to populate into the
# CNI network config.  Otherwise, we'll just fill that in with blanks.
if [ -e "/host/etc/cni/net.d/genie-tls/etcd-ca" ];
then
        CNI_CONF_ETCD_CA=${HOST_SECRETS_DIR}/etcd-ca
fi

if [ -e "/host/etc/cni/net.d/genie-tls/etcd-key" ];
then
        CNI_CONF_ETCD_KEY=${HOST_SECRETS_DIR}/etcd-key
fi

if [ -e "/host/etc/cni/net.d/genie-tls/etcd-cert" ];
then
        CNI_CONF_ETCD_CERT=${HOST_SECRETS_DIR}/etcd-cert
fi

# Place the new binaries if the directory is writeable.
if [ -w "/host/opt/cni/bin/" ]; then
        cp /opt/cni/bin/genie /host/opt/cni/bin/
        echo "Wrote CNIGenie CNI binaries to /host/opt/cni/bin/"
        echo "CNI plugin version: $(/host/opt/cni/bin/genie -v)"
fi

TMP_CONF='/genie.conf.tmp'
# If specified, overwrite the network configuration file.
if [ "${CNI_NETWORK_CONFIG:-}" != "" ]; then
cat >$TMP_CONF <<EOF
${CNI_NETWORK_CONFIG:-}
EOF
fi

# Write a kubeconfig file for the CNI plugin. 
# For now it doesn't use TLS, will be added in future.
cat > /host/etc/cni/net.d/genie-kubeconfig <<EOF
# Kubeconfig file for CNIGenie CNI plugin.
apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    insecure-skip-tls-verify: true
users:
- name: genie
contexts:
- name: genie-context
  context:
    cluster: local
    user: genie
current-context: genie-context
EOF

SERVICEACCOUNT_TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)
sed -i s/__KUBERNETES_SERVICE_HOST__/${KUBERNETES_SERVICE_HOST:-}/g $TMP_CONF
sed -i s/__KUBERNETES_SERVICE_PORT__/${KUBERNETES_SERVICE_PORT:-}/g $TMP_CONF
sed -i s/__KUBERNETES_NODE_NAME__/${KUBERNETES_NODE_NAME:-$(hostname)}/g $TMP_CONF
sed -i s/__SERVICEACCOUNT_TOKEN__/${SERVICEACCOUNT_TOKEN:-}/g $TMP_CONF

#For supporting Romana
sed -i s/__ROMANA_SERVICE_HOST__/${ROMANA_ROOT_SERVICE_HOST:-}/g $TMP_CONF
sed -i s/__ROMANA_SERVICE_PORT__/${ROMANA_ROOT_SERVICE_PORT:-}/g $TMP_CONF

#contains path hence using * instead of /
# NOTWORKING!
#sed -i s*__KUBECONFIG_FILEPATH__*/etc/cni/net.d/genie-kubeconfig*g $TMP_CONF

# Move the temporary CNI config into place.
FILENAME=${CNI_CONF_NAME:-00-genie.conf}
mv $TMP_CONF /host/etc/cni/net.d/${FILENAME}
echo "Wrote CNI config: $(cat /host/etc/cni/net.d/${FILENAME})"

# Unless told otherwise, sleep forever.
# This prevents Kubernetes from restarting the pod repeatedly.
should_sleep=${SLEEP:-"true"}
echo "Done configuring CNI.  Sleep=$should_sleep"
while [ "$should_sleep" == "true"  ]; do
        # Kubernetes Secrets can be updated.  If so, we need to install the updated
        # version to the host. Just check the timestamp on the certificate to see if it
        # has been updated.  A bit hokey, but likely good enough.
        stat_output=$(stat -c%y ${SECRETS_MOUNT_DIR}/etcd-cert 2>/dev/null)
        sleep 10;
        if [ "$stat_output" != "$(stat -c%y ${SECRETS_MOUNT_DIR}/etcd-cert 2>/dev/null)" ]; then
                echo "Updating installed secrets at: $(date)"
                cp ${SECRETS_MOUNT_DIR}/* /host/etc/cni/net.d/genie-tls/
        fi
done

