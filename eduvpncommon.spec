%global name eduvpn-common
%global modname eduvpn_common
%global sum eduVPN common Go library

Name:           lib%{name}
Version:        0.1.0
Release:        1%{?dist}
Summary:        %{sum}

License:        MIT
URL:            https://github.com/jwijenbergh/eduvpn-common
Source0:        %{name}.tar.gz
BuildRequires:  make
BuildRequires:  gcc
BuildRequires:  golang
BuildRequires:  python3-devel
BuildRequires:  python3-setuptools
BuildRequires:  python3-wheel

%description
The client side Go shared library to interact with eduVPN servers

%build
make
pushd wrappers/python
%py3_build
popd

%install
mkdir -p ${RPM_BUILD_ROOT}%_libdir
find exports/lib -name '*.so' -exec mv {} ${RPM_BUILD_ROOT}%_libdir/ \;
pushd wrappers/python
%py3_install
popd

%files
%_libdir/*.so

%prep
%autosetup -n %{name}-%{version}

%package -n python3-%{name}
BuildArch: noarch
Requires: %{name}
Summary: Python3 eduvpncommon wrapper

%description -n python3-%{name}
The python wrapper for the eduVPN common Go shared library

%files -n python3-%{name}
%{python3_sitelib}/%{modname}/
%{python3_sitelib}/%{modname}-%{version}*
