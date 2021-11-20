#!/bin/bash
# Generate testcases with fake keys

# Make sure we do not delete *.minisigs etc. anywhere
if [ ${PWD##*/} != "dummy" ]
then
	>&2 echo "Wrong directory, should be run in dummy/"
	exit 1
fi

rm -f *.minisig *.blake2b

# Uncomment to regenerate keys
#rm -f *.key
#echo -en "\n\n" | minisign -Gf -p public.key -s secret.key &
#echo -en "\n\n" | minisign -Gf -p wrong_public.key -s wrong_secret.key &
#wait

# Try to create pure signature with default Minisign (works with version < 0.10)
echo | minisign -Sm server_list.json -x server_list.json.pure.minisig -t $'timestamp:10\tfile:server_list.json' -s secret.key
# Check if it is actually a prehashed signature
if echo | minisign -VHm server_list.json -x server_list.json.pure.minisig -p public.key
then
	echo "minisign version is >0.9, trying minisign-0.9"
	# If it is, try to sign with some minisign-0.9 program
	if ! echo | minisign-0.9 -Sm server_list.json -x server_list.json.pure.minisig -t $'timestamp:10\tfile:server_list.json' -s secret.key
	then
		>&2 echo -e "\n\nTo produce a non-prehashed signature we need Minisign 0.9\n\n"
	fi
fi

# Rest works with Minisign 0.9 and 0.10 (and up, probably)

echo | minisign -SHm server_list.json -t $'timestamp:10\tfile:server_list.json\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_nohashed.minisig -t $'timestamp:10\tfile:server_list.json' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_latertime.minisig -t $'timestamp:20\tfile:server_list.json\t hashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_orglist.minisig -t $'timestamp:10\tfile:organization_list.json\thashed' -s secret.key &
wait
echo | minisign -SHm server_list.json -x server_list.json.tc_otherfile.minisig -t $'timestamp:10\tfile:otherfile\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_nofile.minisig -t $'timestamp:10\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_notime.minisig -t $'file:server_list.json\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_emptytime.minisig -t $'timestamp:\tfile:server_list.json\thashed' -s secret.key &
wait
echo | minisign -SHm server_list.json -x server_list.json.tc_emptyfile.minisig -t $'timestamp:10\tfile:\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_earliertime.minisig -t $'timestamp:9\tfile:server_list.json\thashed' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.tc_random.minisig -t 'random stuff' -s secret.key &
echo | minisign -SHm server_list.json -x server_list.json.large_time.minisig -t $'timestamp:4300000000\tfile:server_list.json' -s secret.key &
wait

echo | minisign -SHm organization_list.json -t $'timestamp:10\tfile:organization_list.json\thashed' -s secret.key &
echo | minisign -SHm organization_list.json -x organization_list.json.tc_servlist.minisig -t $'timestamp:10\tfile:server_list.json\thashed' -s secret.key &

echo | minisign -SHm other_list.json -t $'timestamp:10\tfile:other_list.json\thashed' -s secret.key &

echo | minisign -SHm server_list.json -x server_list.json.wrong_key.minisig -t $'timestamp:10\tfile:server_list.json\thashed' -s wrong_secret.key &
wait

./generate_forged.py
