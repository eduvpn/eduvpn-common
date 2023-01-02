#!/usr/bin/env python3

import unittest
import eduvpn_common.main as eduvpn
from eduvpn_common.state import State, StateType
import webbrowser
import sys
import os
import json

# Import project root directory where the selenium python utility is
sys.path.append(
    os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
)

from selenium_eduvpn import login_eduvpn


class ConfigTests(unittest.TestCase):
    def testConfig(self):
        _eduvpn = eduvpn.EduVPN("org.letsconnect-vpn.app.linux", "testconfigs", "en")
        # This can throw an exception
        _eduvpn.register()

        @_eduvpn.event.on(State.OAUTH_STARTED, StateType.ENTER)
        def oauth_initialized(old_state, url_json):
            login_eduvpn(url_json)

        server_uri = os.getenv("SERVER_URI")
        if not server_uri:
            print("No SERVER_URI environment variable given, skipping...")
            _eduvpn.deregister()
            return

        # This can throw an exception
        _eduvpn.add_custom_server(server_uri)
        _eduvpn.get_config_custom_server(server_uri)

        # Deregister
        _eduvpn.deregister()

    def testDoubleRegister(self):
        _eduvpn = eduvpn.EduVPN("org.letsconnect-vpn.app.linux", "testconfigs", "en")
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
