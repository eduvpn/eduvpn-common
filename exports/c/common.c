#include "common.h"

void call_callback(PythonCB callback, const char *name, int oldstate, int newstate, void* data)
{
    callback(name, oldstate, newstate, data);
}
