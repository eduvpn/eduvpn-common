package org.eduvpn.common;

/** Signature timestamp smaller than specified minimum signing time (rollback). */
public final class SignatureTooOldException extends VerifyException {
    public SignatureTooOldException() {
        super("replay of previous signature (rollback)");
    }
}
