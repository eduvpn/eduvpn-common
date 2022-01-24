package nl.eduvpn.common;

/** Other unknown error. */
public final class UnknownVerifyException extends VerifyException {
    public UnknownVerifyException(byte code) {
        super(String.format("unknown verify error (%d)", code));
        assert code != 0;
    }
}
