#!/usr/bin/env python3

import unittest
import eduvpncommon.main as eduvpn
import webbrowser
import sys
import os

# Import project root directory where the selenium python utility is
sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))

from selenium_eduvpn import login_eduvpn

class ConfigTests(unittest.TestCase):
    def testConfig(self):
        self._eduvpn = eduvpn.EduVPN("org.eduvpn.app.linux", "testconfigs")
        assert self._eduvpn.register()
        @self._eduvpn.event.on("OAuth_Started", eduvpn.StateType.Enter)
        def oauth_initialized(url):
            login_eduvpn(url)

        server_uri = os.getenv("SERVER_URI")
        if not server_uri:
            self.fail("No SERVER_URI environment variable given")

        config, error = self._eduvpn.get_config_institute_access(server_uri)

        if error != "":
            self.fail(f"Got error: {error} when connecting to {server_uri}")

if __name__ == "__main__":
    unittest.main()
