cmake_minimum_required(VERSION 3.18.1)
project(eduvpn_common)

set(CMAKE_VERBOSE_MAKEFILE on)

# Android -> Go architecture map
set(arch_map_x86	386)
set(arch_map_x86_64	amd64)
set(arch_map_arm	arm)
set(arch_map_arm64	arm64)

set(GOARCH ${arch_map_${ANDROID_ARCH_NAME}})

find_program(MAKE_EXECUTABLE
	NAMES gmake mingw32-make make
	NAMES_PER_DIR
	DOC "GNU Make"
	REQUIRED
)

# Inspired by https://github.com/WireGuard/wireguard-android/blob/1.0.20211029/tunnel/tools/CMakeLists.txt

# --target has to be specified to compiler & linker as e.g. ANDROID_C_COMPILER may just be 'clang' without prefixes
# CGO_CPPFLAGS are concatenated to CGO_CFLAGS and CGO_CXXFLAGS
add_custom_target(shared-lib
	WORKING_DIRECTORY "${CMAKE_CURRENT_SOURCE_DIR}/../../../exports"
	COMMENT "Building shared library for ${ANDROID_LLVM_TRIPLE}"
 	VERBATIM
	COMMAND ${MAKE_EXECUTABLE}
		GOOS=android GOARCH=${GOARCH}
		CC=${ANDROID_C_COMPILER} CXX=${ANDROID_CXX_COMPILER}
		CGO_CPPFLAGS=--target=${ANDROID_LLVM_TRIPLE} CGO_CFLAGS=${CMAKE_C_FLAGS} CGO_CXXFLAGS=${CMAKE_CXX_FLAGS}
		CGO_LDFLAGS=${CMAKE_SHARED_LINKER_FLAGS}\ --target=${ANDROID_LLVM_TRIPLE}
		COPY_LIB_TO=${CMAKE_LIBRARY_OUTPUT_DIRECTORY}
)

# Note about COPY_LIB_TO: this is an easy cross-platform alternative to calling `cp`
# file(COPY ...) does not work since it runs at the configure stage...
