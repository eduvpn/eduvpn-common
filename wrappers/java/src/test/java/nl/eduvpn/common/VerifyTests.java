package nl.eduvpn.common;

import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.time.Instant;

import static org.junit.jupiter.api.Assertions.*;

class VerifyTests {
    private static final Path testDataDir = Paths.get("../../test_data");

    @SuppressWarnings("OptionalGetWithoutIsPresent")
    @BeforeAll
    static void oneTimeSetup() throws IOException {
        Discovery.insecureTestingSetExtraKey(Files.lines(testDataDir.resolve("dummy/public.key")).reduce((a, b) -> b).get());
    }

    @Test
    void testValid() {
        assertDoesNotThrow(() ->
                Discovery.verify(
                        Files.readAllBytes(testDataDir.resolve("dummy/server_list.json.minisig")),
                        Files.readAllBytes(testDataDir.resolve("dummy/server_list.json")),
                        "server_list.json",
                        Instant.EPOCH
                ));
    }

    @Test
    void testInvalidSignature() {
        Assertions.assertEquals(2, assertThrows(VerifyException.class, () ->
                Discovery.verify(
                        Files.readAllBytes(testDataDir.resolve("dummy/random.txt")),
                        Files.readAllBytes(testDataDir.resolve("dummy/server_list.json")),
                        "server_list.json",
                        Instant.EPOCH
                )).code);
    }

    @Test
    void testWrongKey() {
        assertEquals(3, assertThrows(VerifyException.class, () ->
                Discovery.verify(
                        Files.readAllBytes(testDataDir.resolve("dummy/server_list.json.wrong_key.minisig")),
                        Files.readAllBytes(testDataDir.resolve("dummy/server_list.json")),
                        "server_list.json",
                        Instant.EPOCH
                )).code);
    }

    @Test
    void testOldSignature() {
        assertEquals(4, assertThrows(VerifyException.class, () ->
                Discovery.verify(
                        Files.readAllBytes(testDataDir.resolve("dummy/server_list.json.minisig")),
                        Files.readAllBytes(testDataDir.resolve("dummy/server_list.json")),
                        "server_list.json",
                        Instant.MAX
                )).code);
    }

    @Test
    void testUnknownExpectedFile() {
        assertThrows(IllegalArgumentException.class, () ->
                Discovery.verify(
                        Files.readAllBytes(testDataDir.resolve("dummy/other_list.json.minisig")),
                        Files.readAllBytes(testDataDir.resolve("dummy/other_list.json")),
                        "other_list.json",
                        Instant.EPOCH
                ));
    }
}