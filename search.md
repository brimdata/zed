## search hack


```
rm -rf test
setenv ZED_LAKE_ROOT test
zed lake init

# use smaller data objects to make search speed for needle higher
zed lake create -p p1 -S 10MB

zed lake load -p p1 ../zed.SAVE/zed-sample-data/zeek-default/conn.log.gz
1xhL2VhmLDI3b2o84XCjnKTJaSh committed

zed lake index create TEST field id.orig_h
TEST
    rule 1xhL6CriXSzTy604N5VpogEhzRD field id.orig_h

zed lake log -p p1
commit 1xhL2VhmLDI3b2o84XCjnKTJaSh (main)
Author: mccanne@bay.local
Date:   2021-09-05T00:36:06Z

    loaded 9 data objects

    1xhL2EzDQdB3Y3w6edUZBT9stLb 123980 records in 5141097 data bytes  
    1xhL2HpcrnFHmb2tgnVYThEWmTd 123683 records in 5144236 data bytes  
    1xhL2GSG5XqLAi60sOuBhIy68U5 121981 records in 5025907 data bytes  
    1xhL2Q6GXwstTPHCG1ePM4D3hjM 123766 records in 5099557 data bytes  
    1xhL2Ox4XMEeoXJtXtZmyPolSmd 125404 records in 5181475 data bytes  
    1xhL2Mmgus1LJFyT9RKWYGgh65B 124389 records in 5077381 data bytes  
    1xhL2a3vB3ZsYNQUjyX2eDUOl3u 124170 records in 5059101 data bytes  
    1xhL2ZHFXclDzufkplobwffFclY 125232 records in 5069840 data bytes  
    1xhL2V1682FqvnjO2cNMkgsdWb7 29347 records in 1217891 data bytes

zed lake index apply -p p1 TEST  1xhL2EzDQdB3Y3w6edUZBT9stLb
zed lake index apply -p p1 TEST  1xhL2HpcrnFHmb2tgnVYThEWmTd
zed lake index apply -p p1 TEST  1xhL2GSG5XqLAi60sOuBhIy68U5
zed lake index apply -p p1 TEST  1xhL2Q6GXwstTPHCG1ePM4D3hjM
zed lake index apply -p p1 TEST  1xhL2Ox4XMEeoXJtXtZmyPolSmd
zed lake index apply -p p1 TEST  1xhL2Mmgus1LJFyT9RKWYGgh65B
zed lake index apply -p p1 TEST  1xhL2a3vB3ZsYNQUjyX2eDUOl3u
zed lake index apply -p p1 TEST  1xhL2ZHFXclDzufkplobwffFclY
zed lake index apply -p p1 TEST  1xhL2V1682FqvnjO2cNMkgsdWb7

# the ID here is the index rule ID from "zed lake index ls"
 time zed lake query -search "1xhL6CriXSzTy604N5VpogEhzRD:10.47.24.178" "from p1 | id.orig_h==10.47.24.178"
{_path:"conn",ts:2018-03-24T17:26:21.143971Z,uid:"CnxEkC4VNeCLg9I7Yh"(bstring),id:{orig_h:10.47.24.178,orig_p:3(port=(uint16)),resp_h:10.128.0.207,resp_p:1(port)},proto:"icmp"(=zenum),service:null(bstring),duration:5.012219s,orig_bytes:144(uint64),resp_bytes:0(uint64),conn_state:"OTH"(bstring),local_orig:null(bool),local_resp:null(bool),missed_bytes:0(uint64),history:null(bstring),orig_pkts:4(uint64),orig_ip_bytes:256(uint64),resp_pkts:0(uint64),resp_ip_bytes:0(uint64),tunnel_parents:null(|[bstring]|)}
0.112u 0.024s 0:00.04 325.0%	0+0k 0+0io 0pf+0w

# to compare create a pool with a big object threshold (the default is 500MB)

zed lake create -p p2
zed lake load -p p2 ../zed.SAVE/zed-sample-data/zeek-default/conn.log.gz
time zed lake query "from p2 | id.orig_h==10.47.24.178"

{_path:"conn",ts:2018-03-24T17:26:21.143971Z,uid:"CnxEkC4VNeCLg9I7Yh"(bstring),id:{orig_h:10.47.24.178,orig_p:3(port=(uint16)),resp_h:10.128.0.207,resp_p:1(port)},proto:"icmp"(=zenum),service:null(bstring),duration:5.012219s,orig_bytes:144(uint64),resp_bytes:0(uint64),conn_state:"OTH"(bstring),local_orig:null(bool),local_resp:null(bool),missed_bytes:0(uint64),history:null(bstring),orig_pkts:4(uint64),orig_ip_bytes:256(uint64),resp_pkts:0(uint64),resp_ip_bytes:0(uint64),tunnel_parents:null(|[bstring]|)}
0.611u 0.057s 0:00.12 550.0%	0+0k 0+0io 0pf+0w
