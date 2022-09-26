#ifndef ERROR_H
#define ERROR_H

typedef enum errorLevel {
   ERR_OTHER,
   ERR_INFO,
   ERR_WARNING,
   ERR_FATAL,
} errorLevel;

typedef struct error {
   errorLevel level;
   const char* traceback;
   const char* cause;
} error;

#endif /* ERROR_H */
