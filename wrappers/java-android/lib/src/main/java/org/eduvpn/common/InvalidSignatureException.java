package org.eduvpn.common;

/** Signature is invalid (for the expected file type). */
public final class InvalidSignatureException extends VerifyException {
    public InvalidSignatureException() {
        super("invalid signature");
    }
}
