# Package formats

We support the following additional package formats: RPM (Linux, Fedora) and Deb (Linux, Debian derivatives)

# Linux: RPM
To build a RPM, issue the following commands:

```bash
# Make sure we have make
sudo dnf install -y make

# Install dependencies to build RPMs
sudo make rpm-depends

# Create RPM
make rpm
```

This outputs RPMs (the Go library and each wrapper) and a SRPM to the `dist/` folder.

To install these rpms, use ```rpm -i``` where you first install the `libeduvpncommon` RPM and then the associated wrapper. We provide a COPR to make this whole process easy:

```bash
sudo dnf copr enable jwijenbergh/eduvpn-common 

# E.g. install the python wrapper
# This automatically installs the Go shared library ('libeduvpncommon') as well
sudo dnf install python3-eduvpncommon
```

To cross compile manually (without COPR), use the ```make rpm-mock``` target instead of ```make rpm```. This uses the [mock](https://github.com/rpm-software-management/mock) utility which runs each build in a separate environment. The target distro and architecture can be changed with the ```MOCK_TARGET``` flag. E.g:

```bash
# Create a rpm for centos stream 8 aarch64
# You will be prompted for a password if you're not in the 'mock' group
make rpm-mock MOCK_TARGET=centos-stream-8-aarch64
```

A list of targets can be found in ```/etc/mock/*.cfg```. The default target is `fedora-36-x86_64`

# Linux: Deb
