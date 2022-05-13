#!/usr/bin/env python3

import unittest
import eduvpncommon.main as eduvpn
import webbrowser
import sys
import os

# Import project root directory where the selenium python utility is
sys.path.append(
    os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
)

from selenium_eduvpn import login_eduvpn


class ConfigTests(unittest.TestCase):
    def testConfig(self):
        _eduvpn = eduvpn.EduVPN("org.eduvpn.app.linux", "testconfigs")
        # This can throw an exception
        _eduvpn.register()

        @_eduvpn.event.on("OAuth_Started", eduvpn.StateType.Enter)
        def oauth_initialized(old_state, url):
            login_eduvpn(url)

        server_uri = os.getenv("SERVER_URI")
        if not server_uri:
            self.fail("No SERVER_URI environment variable given")

        # This can throw an exception
        _eduvpn.get_config_institute_access(server_uri)

        # Deregister
        _eduvpn.deregister()

    def testDoubleRegister(self):
        _eduvpn = eduvpn.EduVPN("org.eduvpn.app.linux", "testconfigs")
        # This can throw an exception
        _eduvpn.register()
        # This should throw
        try:
            _eduvpn.register()
        except Exception as e:
            return
        self.fail("No exception thrown on second register")


if __name__ == "__main__":
    unittest.main()
