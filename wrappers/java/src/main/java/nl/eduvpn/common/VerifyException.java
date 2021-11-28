package nl.eduvpn.common;

public class VerifyException extends Exception {
    public final long code; //TODO not use plain long

    public VerifyException(long code) {
        this.code = code;
    }
}