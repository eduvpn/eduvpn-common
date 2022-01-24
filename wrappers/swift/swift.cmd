@echo off

:: Rename PATH -> Path because of swift issue https://github.com/compnerd/swift-build/issues/413
set _p=%PATH%
set PATH=
set Path=%_p%
set _p=

swift.exe %*
