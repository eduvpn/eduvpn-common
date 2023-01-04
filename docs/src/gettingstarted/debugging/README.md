# Debugging

To debug this library, e.g. to discover bugs or to see how it works internally, the library comes with a few nice additions.

## The debug variable
To enable debugging, set debugging to True in the method that registers the code with the library (see [API](../api/index.html)). This sets the logging level to `INFO` (meaning show all messages), and generates a Finite State Machine (FSM) `.graph` file. We explain in more detail what these two components (logging and FSM) exactly are and how they can be used.
