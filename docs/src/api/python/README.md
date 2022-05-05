# Python
As the Go library is build as a *shared* library, it can be loaded by other languages. We have created wrapper code for Python to use this library. We define the functions and then give a similar example to the Go example.

The functions that we will discuss are all part of the `EduVPN` class defined in eduvpncommon.main. You can import it like so:

```python
import eduvpncommon.main as eduvpn

# Then use eduvpn.EduVPN
```
