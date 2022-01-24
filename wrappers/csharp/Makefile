.PHONY: build pack test clean

EXPORTS_PATH ?= ../../exports
# Export, see EduVpnCommon.props
export EXPORTS_LIB_PATH ?= $(EXPORTS_PATH)/lib

build:
	dotnet publish EduVpnCommon.csproj --configuration Release

pack:
	dotnet pack EduVpnCommon.csproj --configuration Release

test:
ifneq ($(EXPORTS_PATH),)
ifneq ($(wildcard $(EXPORTS_PATH)/Makefile),)
	$(MAKE) -C "$(EXPORTS_PATH)"
endif
endif
	dotnet test

clean:
	rm -rf bin/ obj/ EduVpnCommonTests/bin/ EduVpnCommonTests/obj/
