<#
.SYNOPSIS
    Generate .lib import library file for specified .dll file.
.NOTES
    Requires dumpbin & lib, may need to execute through VS developer shell.
#>

param (
    [string]$DllPath
)

# Compatible with both Windows PowerShell and PowerShell Core

$ErrorActionPreference = "Stop"

$dll = Get-Item $DllPath
$def = Join-Path $dll.Directory "$( $dll.BaseName ).def"
$lib = Join-Path $dll.Directory "$( $dll.BaseName ).lib"
$machine = (dumpbin /nologo /headers $dll.FullName |
        Select-String -AllMatches 'machine \((.+)\)').Matches[0].Groups[1].Value

"LIBRARY $( $dll.BaseName )`nEXPORTS`n" + (
(dumpbin /nologo /exports $dll.FullName |
        Select-String -AllMatches '\d+\s+\d+\s+[0-9A-Z]+\s+(\S+)').Matches |
        % { $_.Groups[1].Value } |
        where { $_[0] -ne '_' } |  # Skip _cgo_dummy_export
Out-String) |
        Set-Content $def

lib /machine:$machine /def:"$def" /out:"$lib"
