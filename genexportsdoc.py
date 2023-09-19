#!/usr/bin/env python3

import subprocess

cmd=["go", "doc", "--all", "exports"]

output=subprocess.check_output(cmd).decode("utf-8")

section = [""]
package_doc = ""

in_section = True

num = 0
for out in output.splitlines():
    line = out + "\n"
    if line.startswith(" "):
        line = line[4:]
    if out.startswith("FUNCTIONS") or out.startswith("VARIABLES"):
        in_section = False
    if out.startswith("func"):
        in_section = True
        section.append(line)
        num += 1
        continue
    if in_section:
        section[num] += line

def func_name(signature: str) -> str:
    idx = signature.index("(")
    return signature[len("func "):idx]

def gen_toc(title: str) -> str:
    id = "-".join(title.lower().split(" "))
    return f"[{title}](#{id})"

toc = ""
first = True

gen_sections = []
for sec in section:
    if first:
        gen_sections.append(("About the API", sec))
        first = False
        continue
    lines = sec.splitlines()
    signature, doc = lines[0], "\n".join(lines[1:])
    body = f"Signature:\n ```go\n{signature}\n```\n{doc}"
    gen_sections.append((func_name(signature), body))

first = True
toc = "# Table of contents\n"
for title, body in gen_sections:
    if first:
        toc += f"- {gen_toc(title)}\n"
        func_toc = gen_toc("Functions")
        toc += f"- {func_toc}\n"
        first = False
        continue
    toc += f"    * {gen_toc(title)}\n"

data = "This document was automatically generated from the exports/exports.go file\n\n"
data += f"{toc}\n"
first = True
for title, body in gen_sections:
    if first:
        data += f"# {title}\n"
        data += f"{body}\n"
        data += "# Functions\n"
        first = False
        continue
    data += f"## {title}\n"
    data += f"{body}\n"

with open("docs/src/api/functiondocs.md", "w+") as f:
    f.write(data)
