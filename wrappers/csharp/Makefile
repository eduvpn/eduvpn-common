.PHONY: build pack test clean

EXPORTS_PATH ?= ../../exports
include $(EXPORTS_PATH)/common.mk

# Export, see EduVpnCommon.props
export EXPORTS_LIB_PATH

build:
	dotnet publish EduVpnCommon.csproj --configuration Release

pack:
	dotnet pack EduVpnCommon.csproj --configuration Release

test: .try-build-lib
	dotnet test

clean:
	rm -rf bin/ obj/ EduVpnCommonTests/bin/ EduVpnCommonTests/obj/
