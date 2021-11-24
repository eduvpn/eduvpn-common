.PHONY: build pack test clean

build:
	dotnet publish EduVpnCommon.csproj --configuration Release

pack:
	dotnet pack EduVpnCommon.csproj --configuration Release

test:
	$(MAKE) -C ../../exports
	dotnet test

clean:
	rm -rf bin/ obj/ EduVpnCommonTests/bin/ EduVpnCommonTests/obj/
