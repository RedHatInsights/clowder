FROM registry.redhat.io/ubi9/python-312:9.7-1765181854

USER 0

ARG OC_VER=4.20.6

RUN curl -L https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${OC_VER}/openshift-client-linux-${OC_VER}.tar.gz -o /tmp/openshift-client.tar.gz && \
    mkdir -p /tmp/openshift-client && \
    tar -xzf /tmp/openshift-client.tar.gz -C /tmp/openshift-client && \
    install -m 0755 /tmp/openshift-client/oc /usr/local/bin/oc && \
    install -m 0755 /tmp/openshift-client/kubectl /usr/local/bin/kubectl && \
    rm -rf /tmp/openshift-client*

RUN dnf -y install git && dnf clean all && rm -rf /var/cache/dnf

ADD . /opt/app-root/src

RUN pip install --no-cache-dir -r requirements.txt

