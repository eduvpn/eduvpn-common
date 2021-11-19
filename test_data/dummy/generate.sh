#!/bin/bash
# Generate testcases with fake keys

# Make sure we do not delete *.minisigs etc. anywhere
if [ ${PWD##*/} != "dummy" ]
then
	>&2 echo "Wrong directory, should be run in dummy/"
	exit 1
fi

rm -f *.minisig *.blake2b *.key

echo -en "\n\n" | minisign -Gf -p public.key -s secret.key &
echo -en "\n\n" | minisign -Gf -p wrong_public.key -s wrong_secret.key

# Try to create pure signature with default Minisign (works with version < 0.10)
echo | minisign -Sm server_list.json -x server_list.json.pure.minisig -t $'time:10\tfile:server_list.json' -s secret.key
# Check if it is actually a prehashed signature
if echo | minisign -VHm server_list.json -x server_list.json.pure.minisig -p public.key
then
	echo "minisign version is >0.9, trying minisign-0.9"
	# If it is, try to sign with some minisign-0.9 program
	if ! echo | minisign-0.9 -Sm server_list.json -x server_list.json.pure.minisig -t $'time:10\tfile:server_list.json' -s secret.key
	then
		>&2 echo -e "\n\nTo produce a non-prehashed signature we need Minisign 0.9\n\n"
	fi
fi

# Rest works with Minisign 0.9 and 0.10 (and up, probably)

echo | minisign -SHm server_list.json -t $'time:10\tfile:server_list.json\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_nohashed.minisig -t $'time:10\tfile:server_list.json' -s secret.key
echo | minisign -SHm server_list.json -x server_list.json.tc_latertime.minisig -t $'time:20\tfile:server_list.json\t hashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_orglist.minisig -t $'time:10\tfile:organization_list.json\thashed' -s secret.key
echo | minisign -SHm server_list.json -x server_list.json.tc_otherfile.minisig -t $'time:10\tfile:otherfile\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_nofile.minisig -t $'time:10\thashed' -s secret.key
echo | minisign -SHm server_list.json -x server_list.json.tc_notime.minisig -t $'file:server_list.json\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_empty.minisig -t '' -s secret.key
echo | minisign -SHm server_list.json -x server_list.json.tc_emptytime.minisig -t $'time:\tfile:server_list.json\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_emptyfile.minisig -t $'time:10\tfile:\thashed' -s secret.key
echo | minisign -SHm server_list.json -x server_list.json.tc_earliertime.minisig -t $'time:9\tfile:server_list.json\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_random.minisig -t 'random stuff' -s secret.key
echo | minisign -SHm server_list-large_time.json -x server_list.json.large_time.minisig -t $'time:4300000000\tfile:server_list.json' -s secret.key &
echo | minisign -SHm server_list-no_version.json -x server_list.json.no_version.minisig -t $'time:10\tfile:server_list.json\thashed' -s secret.key

echo | minisign -SHm organization_list.json -t $'time:10\tfile:organization_list.json\thashed' -s secret.key &
echo | minisign -SHm organization_list.json -x organization_list.json.tc_servlist.minisig -t $'time:10\tfile:server_list.json\thashed' -s secret.key

echo | minisign -SHm other_list.json -t $'time:10\tfile:other_list.json\thashed' -s secret.key &
echo | minisign -SHm other_list.json -x other_list.json.tc_servlist.minisig -t $'time:10\tfile:server_list.json\thashed' -s secret.key
echo | minisign -SHm no_list.json -t $'time:10\tfile:server_list.json\thashed' -s secret.key &
echo | minisign -SHm random.txt -t $'time:10\tfile:server_list.json\thashed' -s secret.key
echo | minisign -SHm empty -t $'time:10\tfile:server_list.json\thashed' -s secret.key &

echo | minisign -SHm wrong_type1.json -t $'time:10\tfile:server_list.json\thashed' -s secret.key &
echo | minisign -SHm wrong_type2.json -t $'time:10\tfile:server_list.json\thashed' -s secret.key
echo | minisign -SHm wrong_type3.json -t $'time:10\tfile:server_list.json\thashed' -s secret.key &

echo | minisign -SHm server_list.json -x server_list.json.wrong_key.minisig -t $'time:10\tfile:server_list.json\thashed' -s wrong_secret.key

./generate_forged.py
