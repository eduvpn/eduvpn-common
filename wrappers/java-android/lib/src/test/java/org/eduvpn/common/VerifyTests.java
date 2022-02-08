package org.eduvpn.common;

import org.apache.commons.io.IOUtils;
import org.junit.BeforeClass;
import org.junit.Test;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;

public class VerifyTests {
    private static byte[] readAll(String resource) throws IOException {
        try (InputStream stream = VerifyTests.class.getResourceAsStream(resource)) {
            return IOUtils.toByteArray(stream);
        }
    }

    @SuppressWarnings("OptionalGetWithoutIsPresent")
    @BeforeClass
    public static void oneTimeSetup() throws IOException {
        try (BufferedReader reader = new BufferedReader(new InputStreamReader(
                VerifyTests.class.getResourceAsStream("public.key")))) {
            Discovery.insecureTestingSetExtraKey(reader.lines().reduce((a, b) -> b).get());
        }
    }

    @Test
    public void testValid() throws IOException, VerifyException {
        Discovery.verify(
                readAll("server_list.json.minisig"),
                readAll("server_list.json"),
                "server_list.json",
                0
        );
    }

    @Test(expected = InvalidSignatureException.class)
    public void testInvalidSignature() throws IOException, VerifyException {
        Discovery.verify(
                readAll("random.txt"),
                readAll("server_list.json"),
                "server_list.json",
                0
        );
    }

    @Test(expected = InvalidSignatureUnknownKeyException.class)
    public void testWrongKey() throws IOException, VerifyException {
        Discovery.verify(
                readAll("server_list.json.wrong_key.minisig"),
                readAll("server_list.json"),
                "server_list.json",
                0
        );
    }

    @Test(expected = SignatureTooOldException.class)
    public void testOldSignature() throws IOException, VerifyException {
        Discovery.verify(
                readAll("server_list.json.minisig"),
                readAll("server_list.json"),
                "server_list.json",
                Long.MAX_VALUE
        );
    }

    @Test(expected = IllegalArgumentException.class)
    public void testUnknownExpectedFile() throws IOException, VerifyException {
        Discovery.verify(
                readAll("other_list.json.minisig"),
                readAll("other_list.json"),
                "other_list.json",
                0
        );
    }
}
