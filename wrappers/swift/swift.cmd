@echo off

:: Rename PATH -> Path
set _p=%PATH%
set PATH=
set Path=%_p%
set _p=

swift.exe %*
