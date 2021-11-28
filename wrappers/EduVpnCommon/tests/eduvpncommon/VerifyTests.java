package eduvpncommon;

import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.time.Instant;

class VerifyTests {
    private static final Path testDataDir = Paths.get("../../test_data");

    @BeforeAll
    static void oneTimeSetup() throws IOException {
        Discovery.insecureTestingSetExtraKey(Files.lines(testDataDir.resolve("dummy/public.key")).reduce((a, b) -> b).get());
    }

    @Test
    void testValid() throws IOException, VerifyException {
        Discovery.verify(
                Files.readAllBytes(Paths.get("../../test_data/dummy/server_list.json.minisig")),
                Files.readAllBytes(Paths.get("../../test_data/dummy/server_list.json")),
                "server_list.json",
                Instant.EPOCH
        );
    }

    //TODO
}