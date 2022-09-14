typedef void (*PythonCB)(const char* name, int oldstate, int newstate, void* data);

void call_callback(PythonCB callback, const char *name, int oldstate, int newstate, void* data);
