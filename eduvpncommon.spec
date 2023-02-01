%global libname eduvpn-common
%global modname eduvpn_common
%global sum eduVPN common Go library

Name:           lib%{libname}
Version:        0.3.0
Release:        0.1%{?dist}
Summary:        %{sum}

License:        MIT
URL:            https://github.com/eduvpn/eduvpn-common
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

%package -n python3-%{libname}
BuildArch: noarch
Requires: %{name} >= 0.3.0, %{name} < 0.4.0
Summary: Python3 eduvpncommon wrapper

%description -n python3-%{libname}
The python wrapper for the eduVPN common Go shared library

%files -n python3-%{libname}
%{python3_sitelib}/%{modname}/
%{python3_sitelib}/%{modname}-%{version}*
