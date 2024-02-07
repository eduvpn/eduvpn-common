#!/usr/bin/env python3

import unittest
import eduvpn_common.main as eduvpn
import eduvpn_common.event as event
from eduvpn_common.state import State, StateType
import os
import sys
import threading

# Import project root directory where the selenium python utility is
sys.path.append(
    os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
)

from selenium_eduvpn import login_eduvpn


class Handler:
    @event.class_state_transition(State.OAUTH_STARTED, StateType.ENTER)
    def on_oauth(self, old_state: State, data: str):
        t1 = threading.Thread(target=login_eduvpn, args=(data,))
        t1.start()


class ConfigTests(unittest.TestCase):
    def testConfig(self):
        _eduvpn = eduvpn.EduVPN("org.letsconnect-vpn.app.linux", "0.1.0", "testconfigs")
        # This can throw an exception
        _eduvpn.register()
        handler = Handler()
        _eduvpn.register_class_callbacks(handler)

        server_uri = os.getenv("SERVER_URI")
        if not server_uri:
            print("No SERVER_URI environment variable given, skipping...")
            _eduvpn.deregister()
            return

        # This can throw an exception
        _eduvpn.add_server(eduvpn.ServerType.CUSTOM, server_uri)
        _eduvpn.get_config(eduvpn.ServerType.CUSTOM, server_uri)

        # Deregister
        _eduvpn.deregister()

    def testDoubleRegister(self):
        _eduvpn = eduvpn.EduVPN("org.letsconnect-vpn.app.linux", "0.1.0", "testconfigs")
        # This can throw an exception
        _eduvpn.register()
        handler = Handler()
        _eduvpn.register_class_callbacks(handler)
        # This should throw
        try:
            _eduvpn.register()
        except Exception as e:
            return
        self.fail("No exception thrown on second register")


if __name__ == "__main__":
    unittest.main()
