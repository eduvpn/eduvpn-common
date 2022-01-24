package nl.eduvpn.common;

import com.sun.jna.*;

import java.nio.charset.StandardCharsets;
import java.time.Instant;

public final class Discovery {
    private static final String LIB_NAME = "eduvpn_common";
    private static final NativeApi discovery = Native.load(LIB_NAME, NativeApi.class);

    /**
     * Verifies the signature on the JSON server_list.json/organization_list.json file.
     * If the function returns, the signature is valid for the given file type.
     *
     * @param signature        .minisig signature file contents.
     * @param signedJson       Signed .json file contents.
     * @param expectedFileName The file type to be verified, one of {@code "server_list.json"} or {@code "organization_list.json"}.
     * @param minSignTime      Minimum time for signature. Should be set to at least the time of the previous signature.
     * @throws IllegalArgumentException If {@code expectedFileName} is not one of the allowed values or one of the parameters is empty.
     * @throws VerifyException          If signature verification fails.
     */
    public static void verify(byte[] signature, byte[] signedJson, String expectedFileName, Instant minSignTime) throws VerifyException {
        byte err = discovery.Verify(NativeApi.GoSlice.fromArray(signature), NativeApi.GoSlice.fromArray(signedJson),
                NativeApi.GoSlice.fromString(expectedFileName), minSignTime.getEpochSecond());

        switch (err) {
            case 0:
                return;
            case 1:
                throw new IllegalArgumentException("unknown expected file name");
            case 2:
                throw new InvalidSignatureException();
            case 3:
                throw new InvalidSignatureUnknownKeyException();
            case 4:
                throw new SignatureTooOldException();
            default:
                throw new UnknownVerifyException(err);
        }
    }

    /** Use for testing only, see Go documentation. */
    /*package-private*/
    static void insecureTestingSetExtraKey(String keyString) {
        discovery.InsecureTestingSetExtraKey(NativeApi.GoSlice.fromArray(keyString.getBytes(StandardCharsets.UTF_8)));
    }

    private interface NativeApi extends Library {
        // C-compatible structure
        @Structure.FieldOrder({"data", "len", "cap"})
        class GoSlice extends Structure implements Structure.ByValue {
            public Pointer data;
            public long len, cap;

            public GoSlice(Pointer data, long len, long cap) {
                this.data = data;
                this.len = len;
                this.cap = cap;
            }

            public static GoSlice fromArray(byte[] bytes) {
                Memory memory = new Memory(bytes.length);
                memory.write(0, bytes, 0, bytes.length);
                return new GoSlice(memory, bytes.length, bytes.length);
            }

            /** From string as UTF-8. */
            public static GoSlice fromString(String str) {
                return fromArray(str.getBytes(StandardCharsets.UTF_8));
            }
        }

        byte Verify(GoSlice signatureFileContent, GoSlice signedJson, GoSlice expectedFileName, long minSignTime);

        void InsecureTestingSetExtraKey(GoSlice keyString);
    }

    private Discovery() {
    }
}
