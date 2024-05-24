#!/usr/bin/env python3

import hashlib
import base64

# Hash server_list.json

with open("server_list.json", "rb") as f:
    b = f.read()

with open("server_list.json.blake2b", "wb") as f:
    f.write(hashlib.blake2b(b).digest())

# Forge pure signature on hash, see https://github.com/jedisct1/minisign/issues/104

with open("server_list.json.minisig", "rb") as f:
    siglines = f.readlines()

siglines[0] = b"untrusted comment: this signature has ED changed to Ed\n"
sig = base64.b64decode(siglines[1])
siglines[1] = base64.b64encode(b"Ed" + sig[2:]) + b"\n"

with open("server_list.json.forged_pure.minisig", "wb") as f:
    f.writelines(siglines)
    # Should now work: minisign -Vm server_list.json.blake2b -x server_list.json.forged_pure.minisig -p public-key

# Try to forge key ID

with open("server_list.json.wrong_key.minisig", "rb") as f:
    siglines = f.readlines()

siglines[0] = (
    b"untrusted comment: this signature was created with wrong_secret.key but has key ID changed to that of public.key\n"
)
sig_wrong = base64.b64decode(siglines[1])
siglines[1] = (
    base64.b64encode(sig_wrong[:2] + sig[2 : 2 + 8] + sig_wrong[2 + 8 :]) + b"\n"
)

with open("server_list.json.forged_keyid.minisig", "wb") as f:
    f.writelines(siglines)
