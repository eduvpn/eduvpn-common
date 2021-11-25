#!/usr/bin/env python3

import unittest
import discovery

test_data_dir = "../../test_data"


def read_bytes(path: str) -> bytes:
    with open(path, "rb") as f:
        return f.read()


class VerifyTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        with open(f"{test_data_dir}/dummy/public.key") as f:
            discovery._insecure_testing_set_extra_key(f.readlines()[-1][:-1])

    def testValid(self):
        discovery.verify(
            read_bytes(f"{test_data_dir}/dummy/server_list.json.minisig"),
            read_bytes(f"{test_data_dir}/dummy/server_list.json"),
            "server_list.json",
            0
        )

    def testValidMemoryView(self):
        discovery.verify(
            memoryview(b"abc" + read_bytes(f"{test_data_dir}/dummy/server_list.json.minisig") + b"abc")[3:-3],
            read_bytes(f"{test_data_dir}/dummy/server_list.json"),
            "server_list.json",
            0
        )

    def testInvalidSignature(self):
        with self.assertRaises(discovery.VerifyError) as ctx:
            discovery.verify(
                read_bytes(f"{test_data_dir}/dummy/random.txt"),
                read_bytes(f"{test_data_dir}/dummy/server_list.json"),
                "server_list.json",
                0
            )
        self.assertEqual(ctx.exception.code, discovery.VerifyErrorCode.ErrInvalidSignature)

    def testWrongKey(self):
        with self.assertRaises(discovery.VerifyError) as ctx:
            discovery.verify(
                read_bytes(f"{test_data_dir}/dummy/server_list.json.wrong_key.minisig"),
                read_bytes(f"{test_data_dir}/dummy/server_list.json"),
                "server_list.json",
                0
            )
        self.assertEqual(ctx.exception.code, discovery.VerifyErrorCode.ErrInvalidSignatureUnknownKey)

    def testOldSignature(self):
        with self.assertRaises(discovery.VerifyError) as ctx:
            discovery.verify(
                read_bytes(f"{test_data_dir}/dummy/server_list.json.minisig"),
                read_bytes(f"{test_data_dir}/dummy/server_list.json"),
                "server_list.json",
                1 << 31
            )
        self.assertEqual(ctx.exception.code, discovery.VerifyErrorCode.ErrTooOld)

    def TestUnknownExpectedFile(self):
        with self.assertRaises(discovery.VerifyError) as ctx:
            discovery.verify(
                read_bytes(f"{test_data_dir}/dummy/other_list.json.minisig"),
                read_bytes(f"{test_data_dir}/dummy/other_list.json"),
                "other_list.json",
                0
            )
        self.assertEqual(ctx.exception.code, discovery.VerifyErrorCode.ErrUnknownExpectedFileName)


if __name__ == "__main__":
    unittest.main()
