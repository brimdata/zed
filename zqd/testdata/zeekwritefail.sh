#!/usr/bin/env bash

# Simulates a zeek run where a log data is written and then zeek fails
# afterwards.

cat <<EOF > conn.log
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#path	conn
#open	2019-11-08-11-44-16
#fields	ts	uid	id.orig_h	id.orig_p	id.resp_h	id.resp_p	proto	service	duration	orig_bytes	resp_bytes	conn_state	local_orig	local_resp	missed_bytes	history	orig_pkts	orig_ip_bytes	resp_pkts	resp_ip_bytes	tunnel_parents
#types	time	string	addr	port	addr	port	enum	string	interval	count	count	string	bool	bool	count	string	count	count	count	count	set[string]
1521911721.255387	C8Tful1TvM3Zf5x8fl	10.164.94.120	39681	10.47.3.155	3389	tcp	-	0.004266	97	19	RSTR	-	-	0	ShADTdtr	10	730	6	342	-
1521911721.411148	CXWfTK3LRdiuQxBbM6	10.47.25.80	50817	10.128.0.218	23189	tcp	-	0.000486	0	0	REJ	-	-	0	Sr	2	104	2	80	-
1521911721.926018	CM59GGQhNEoKONb5i	10.47.25.80	50817	10.128.0.218	23189	tcp	-	0.000538	0	0	REJ	-	-	0	Sr	2	104	2	80	-
1521911722.690601	CuKFds250kxFgkhh8f	10.47.25.80	50813	10.128.0.218	27765	tcp	-	0.000546	0	0	REJ	-	-	0	Sr	2	104	2	80	-
1521911723.205187	CBrzd94qfowOqJwCHa	10.47.25.80	50813	10.128.0.218	27765	tcp	-	0.000605	0	0	REJ	-	-	0	Sr	2	104	2	80	-
1521911724.896854	CFzn9A3l9ppbMBVin3	10.164.94.120	40659	10.47.8.208	3389	tcp	-	0.011922	147	19	RSTR	-	-	0	ShADTdtr	10	830	6	342	-
EOF

exit 1
