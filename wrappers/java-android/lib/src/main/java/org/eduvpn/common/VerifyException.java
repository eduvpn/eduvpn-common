package org.eduvpn.common;

/** Verification failed, do not trust the file. */
public abstract class VerifyException extends Exception {
    protected VerifyException(String message) {
        super(message);
    }
}
