# Logging
As said, logging is used. The logging gets saved in a client-specified directory (see [API](../../api/index.html)). Logging has the following levels:

- `INFO`: Messages purely for info, these do not indicate any errors. They are merely there for debugging purposes
- `WARNING`: These messages indicate a warning, e.g. non-fatal errors
- `ERROR`: Fatal errors which refuses the app from working correctly

By default only messages below or equal to `WARNING` are logged (`WARNING`, `ERROR`). However, if the debug variable is set to `True`, all messages will be logged into the log file.
