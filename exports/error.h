#ifndef ERROR_H
#define ERROR_H

typedef struct error {
   const char* traceback;
   const char* cause;
} error;

#endif /* ERROR_H */
