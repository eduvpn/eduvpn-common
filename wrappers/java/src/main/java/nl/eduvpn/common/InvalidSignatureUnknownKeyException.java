package nl.eduvpn.common;

/** Signature was created with an unknown key and has not been verified. */
public final class InvalidSignatureUnknownKeyException extends VerifyException {
    public InvalidSignatureUnknownKeyException() {
        super("invalid signature (unknown key)");
    }
}
