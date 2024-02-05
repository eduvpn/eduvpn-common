FROM quay.io/pypa/manylinux_2_28_aarch64

RUN yum install -y golang python3 python3-setuptools python3-wheel wget

WORKDIR /pip

ARG COMMONVERSION

RUN wget -O eduvpn-common.tar.xz https://github.com/eduvpn/eduvpn-common/releases/download/$COMMONVERSION/eduvpn-common-$COMMONVERSION.tar.xz
RUN tar xf eduvpn-common.tar.xz

WORKDIR /pip/eduvpn-common-$COMMONVERSION

RUN CGO_ENABLED=1 go build -buildvcs=false -o lib/linux/arm64/libeduvpn_common-$COMMONVERSION.so -buildmode=c-shared ./exports

WORKDIR /pip/eduvpn-common-$COMMONVERSION/wrappers/python

RUN ./setup.py bdist_wheel --exports-lib-path="../../lib"

RUN auditwheel repair dist/*

RUN mkdir /wheelhouse
RUN cp -r wheelhouse/* /wheelhouse