# About
This chapter contains background information for the library. We give a general introduction to eduVPN and explain what problems this library aims to solve.

## eduVPN introduction
eduVPN-common is a library for [eduVPN](https://www.eduvpn.org/), which is a VPN by [Surf](https://www.surf.nl) for research institutions such as Universities. Each institution that  uses eduVPN has its own server. To discover these servers and establish a VPN connection with them, eduVPN clients are used. eduVPN has clients for each common platform:
- [Android](https://github.com/eduvpn/android)
- [Linux](https://github.com/eduvpn/python-eduvpn-client)
- [MacOS/iOS](https://github.com/eduvpn/apple)
- [Windows](https://github.com/Amebis/eduVPN)

## The problem
However, as these clients are rather similar in functionality, apart from platform specific differences, right now there is duplicate code between them. For example, the process to discover institution's servers, the authorization process (OAuth) and Wireguard key generation.
This goal of this library is to provide the common functionality between these clients into one codebase. The library is written in the [Go](https://go.dev/) language and has wrapper code for each of the languages that are used by the current clients.

## Authors
This library is written by [Steven Wallis de Vries](https://github.com/stevenwdv) and [Jeroen Wijenbergh](https://github.com/jwijenbergh), two Radboud University students that worked at Surf for their research internship.
