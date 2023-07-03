---
sidebar_position: 3
sidebar_label: Shaping Zeek NDJSON
---

# Shaping Zeek NDJSON

As described in [Reading Zeek Log Formats](reading-zeek-log-formats.md),
logs output by Zeek in NDJSON format lose much of their rich data typing that
was originally present inside Zeek. This detail can be restored using a Zed
shaper, such as the reference `shaper.zed` described below.

A full description of all that's possible with shapers is beyond the scope of
this doc. However, this example for shaping Zeek NDJSON is quite simple and
is described below.

## Zeek Version/Configuration

The fields and data types in the reference `shaper.zed` reflect the default
NDJSON-format logs output by Zeek releases up to the version number referenced
in the comments at the top of that file. They have been revisited periodically
as new Zeek versions have been released.

Most changes we've observed in Zeek logs between versions have involved only the
addition of new fields. Because of this, we expect the shaper should be usable
as is for Zeek releases older than the one most recently tested, since fields
in the shaper not present in your environment would just be filled in with
`null` values.

[Zeek v4.1.0](https://github.com/zeek/zeek/releases/tag/v4.1.0) is the first
release we've seen since starting to maintain this reference shaper where
field names for the same log type have _changed_ between releases. Because
of this, as shown below, the shaper includes `switch` logic that applies
different type definitions based on the observed field names that are known
to be specific to newer Zeek releases.

All attempts will be made to update this reference shaper in a timely manner
as new Zeek versions are released. However, if you have modified your Zeek
installation with [packages](https://packages.zeek.org/)
or other customizations, or if you are using a [Corelight Sensor](https://corelight.com/products/appliance-sensors/)
that produces Zeek logs with many fields and logs beyond those found in open
source Zeek, the reference shaper will not cover all the fields in your logs.
[As described below](#zed-pipeline), the reference shaper will assign
inferred types to such additional fields. By exploring your data, you can then
iteratively enhance your shaper to match your environment. If you need
assistance, please speak up on our [public Slack](https://www.brimdata.io/join-slack/).

## Reference Shaper Contents

The following reference `shaper.zed` may seem large, but ultimately it follows a
fairly simple pattern that repeats across the many [Zeek log types](https://docs.zeek.org/en/master/script-reference/log-files.html).

```mdtest-input shaper.zed
// This reference Zed shaper for Zeek NDJSON logs was most recently tested with
// Zeek v4.1.0. The fields and data types reflect the default NDJSON
// logs output by that Zeek version when using the JSON Streaming Logs package.

type port=uint16
type zenum=string
type conn_id={orig_h:ip,orig_p:port,resp_h:ip,resp_p:port}

// This first block of type definitions covers the fields we've observed in
// an out-of-the-box Zeek v4.0.3 and earlier, as well as all out-of-the-box
// Zeek v4.1.0 log types except for "ssl" and "x509".

type broker={_path:string,ts:time,ty:zenum,ev:string,peer:{address:string,bound_port:port},message:string,_write_ts:time}
type capture_loss={_path:string,ts:time,ts_delta:duration,peer:string,gaps:uint64,acks:uint64,percent_lost:float64,_write_ts:time}
type cluster={_path:string,ts:time,node:string,message:string,_write_ts:time}
type config={_path:string,ts:time,id:string,old_value:string,new_value:string,location:string,_write_ts:time}
type conn={_path:string,ts:time,uid:string,id:conn_id,proto:zenum,service:string,duration:duration,orig_bytes:uint64,resp_bytes:uint64,conn_state:string,local_orig:bool,local_resp:bool,missed_bytes:uint64,history:string,orig_pkts:uint64,orig_ip_bytes:uint64,resp_pkts:uint64,resp_ip_bytes:uint64,tunnel_parents:|[string]|,_write_ts:time}
type dce_rpc={_path:string,ts:time,uid:string,id:conn_id,rtt:duration,named_pipe:string,endpoint:string,operation:string,_write_ts:time}
type dhcp={_path:string,ts:time,uids:|[string]|,client_addr:ip,server_addr:ip,mac:string,host_name:string,client_fqdn:string,domain:string,requested_addr:ip,assigned_addr:ip,lease_time:duration,client_message:string,server_message:string,msg_types:[string],duration:duration,_write_ts:time}
type dnp3={_path:string,ts:time,uid:string,id:conn_id,fc_request:string,fc_reply:string,iin:uint64,_write_ts:time}
type dns={_path:string,ts:time,uid:string,id:conn_id,proto:zenum,trans_id:uint64,rtt:duration,query:string,qclass:uint64,qclass_name:string,qtype:uint64,qtype_name:string,rcode:uint64,rcode_name:string,AA:bool,TC:bool,RD:bool,RA:bool,Z:uint64,answers:[string],TTLs:[duration],rejected:bool,_write_ts:time}
type dpd={_path:string,ts:time,uid:string,id:conn_id,proto:zenum,analyzer:string,failure_reason:string,_write_ts:time}
type files={_path:string,ts:time,fuid:string,tx_hosts:|[ip]|,rx_hosts:|[ip]|,conn_uids:|[string]|,source:string,depth:uint64,analyzers:|[string]|,mime_type:string,filename:string,duration:duration,local_orig:bool,is_orig:bool,seen_bytes:uint64,total_bytes:uint64,missing_bytes:uint64,overflow_bytes:uint64,timedout:bool,parent_fuid:string,md5:string,sha1:string,sha256:string,extracted:string,extracted_cutoff:bool,extracted_size:uint64,_write_ts:time}
type ftp={_path:string,ts:time,uid:string,id:conn_id,user:string,password:string,command:string,arg:string,mime_type:string,file_size:uint64,reply_code:uint64,reply_msg:string,data_channel:{passive:bool,orig_h:ip,resp_h:ip,resp_p:port},fuid:string,_write_ts:time}
type http={_path:string,ts:time,uid:string,id:conn_id,trans_depth:uint64,method:string,host:string,uri:string,referrer:string,version:string,user_agent:string,origin:string,request_body_len:uint64,response_body_len:uint64,status_code:uint64,status_msg:string,info_code:uint64,info_msg:string,tags:|[zenum]|,username:string,password:string,proxied:|[string]|,orig_fuids:[string],orig_filenames:[string],orig_mime_types:[string],resp_fuids:[string],resp_filenames:[string],resp_mime_types:[string],_write_ts:time}
type intel={_path:string,ts:time,uid:string,id:conn_id,seen:{indicator:string,indicator_type:zenum,where:zenum,node:string},matched:|[zenum]|,sources:|[string]|,fuid:string,file_mime_type:string,file_desc:string,_write_ts:time}
type irc={_path:string,ts:time,uid:string,id:conn_id,nick:string,user:string,command:string,value:string,addl:string,dcc_file_name:string,dcc_file_size:uint64,dcc_mime_type:string,fuid:string,_write_ts:time}
type kerberos={_path:string,ts:time,uid:string,id:conn_id,request_type:string,client:string,service:string,success:bool,error_msg:string,from:time,till:time,cipher:string,forwardable:bool,renewable:bool,client_cert_subject:string,client_cert_fuid:string,server_cert_subject:string,server_cert_fuid:string,_write_ts:time}
type known_certs={_path:string,ts:time,host:ip,port_num:port,subject:string,issuer_subject:string,serial:string,_write_ts:time}
type known_hosts={_path:string,ts:time,host:ip,_write_ts:time}
type known_services={_path:string,ts:time,host:ip,port_num:port,port_proto:zenum,service:|[string]|,_write_ts:time}
type loaded_scripts={_path:string,name:string,_write_ts:time}
type modbus={_path:string,ts:time,uid:string,id:conn_id,func:string,exception:string,_write_ts:time}
type mysql={_path:string,ts:time,uid:string,id:conn_id,cmd:string,arg:string,success:bool,rows:uint64,response:string,_write_ts:time}
type netcontrol={_path:string,ts:time,rule_id:string,category:zenum,cmd:string,state:zenum,action:string,target:zenum,entity_type:string,entity:string,mod:string,msg:string,priority:int64,expire:duration,location:string,plugin:string,_write_ts:time}
type netcontrol_drop={_path:string,ts:time,rule_id:string,orig_h:ip,orig_p:port,resp_h:ip,resp_p:port,expire:duration,location:string,_write_ts:time}
type netcontrol_shunt={_path:string,ts:time,rule_id:string,f:{src_h:ip,src_p:port,dst_h:ip,dst_p:port},expire:duration,location:string,_write_ts:time}
type notice={_path:string,ts:time,uid:string,id:conn_id,fuid:string,file_mime_type:string,file_desc:string,proto:zenum,note:zenum,msg:string,sub:string,src:ip,dst:ip,p:port,n:uint64,peer_descr:string,actions:|[zenum]|,email_dest:|[string]|,suppress_for:duration,remote_location:{country_code:string,region:string,city:string,latitude:float64,longitude:float64},_write_ts:time}
type notice_alarm={_path:string,ts:time,uid:string,id:conn_id,fuid:string,file_mime_type:string,file_desc:string,proto:zenum,note:zenum,msg:string,sub:string,src:ip,dst:ip,p:port,n:uint64,peer_descr:string,actions:|[zenum]|,email_dest:|[string]|,suppress_for:duration,remote_location:{country_code:string,region:string,city:string,latitude:float64,longitude:float64},_write_ts:time}
type ntlm={_path:string,ts:time,uid:string,id:conn_id,username:string,hostname:string,domainname:string,server_nb_computer_name:string,server_dns_computer_name:string,server_tree_name:string,success:bool,_write_ts:time}
type ntp={_path:string,ts:time,uid:string,id:conn_id,version:uint64,mode:uint64,stratum:uint64,poll:duration,precision:duration,root_delay:duration,root_disp:duration,ref_id:string,ref_time:time,org_time:time,rec_time:time,xmt_time:time,num_exts:uint64,_write_ts:time}
type ocsp={_path:string,ts:time,id:string,hashAlgorithm:string,issuerNameHash:string,issuerKeyHash:string,serialNumber:string,certStatus:string,revoketime:time,revokereason:string,thisUpdate:time,nextUpdate:time,_write_ts:time}
type openflow={_path:string,ts:time,dpid:uint64,match:{in_port:uint64,dl_src:string,dl_dst:string,dl_vlan:uint64,dl_vlan_pcp:uint64,dl_type:uint64,nw_tos:uint64,nw_proto:uint64,nw_src:net,nw_dst:net,tp_src:uint64,tp_dst:uint64},flow_mod:{cookie:uint64,table_id:uint64,command:zenum=string,idle_timeout:uint64,hard_timeout:uint64,priority:uint64,out_port:uint64,out_group:uint64,flags:uint64,actions:{out_ports:[uint64],vlan_vid:uint64,vlan_pcp:uint64,vlan_strip:bool,dl_src:string,dl_dst:string,nw_tos:uint64,nw_src:ip,nw_dst:ip,tp_src:uint64,tp_dst:uint64}},_write_ts:time}
type packet_filter={_path:string,ts:time,node:string,filter:string,init:bool,success:bool,_write_ts:time}
type pe={_path:string,ts:time,id:string,machine:string,compile_ts:time,os:string,subsystem:string,is_exe:bool,is_64bit:bool,uses_aslr:bool,uses_dep:bool,uses_code_integrity:bool,uses_seh:bool,has_import_table:bool,has_export_table:bool,has_cert_table:bool,has_debug_data:bool,section_names:[string],_write_ts:time}
type radius={_path:string,ts:time,uid:string,id:conn_id,username:string,mac:string,framed_addr:ip,tunnel_client:string,connect_info:string,reply_msg:string,result:string,ttl:duration,_write_ts:time}
type rdp={_path:string,ts:time,uid:string,id:conn_id,cookie:string,result:string,security_protocol:string,client_channels:[string],keyboard_layout:string,client_build:string,client_name:string,client_dig_product_id:string,desktop_width:uint64,desktop_height:uint64,requested_color_depth:string,cert_type:string,cert_count:uint64,cert_permanent:bool,encryption_level:string,encryption_method:string,_write_ts:time}
type reporter={_path:string,ts:time,level:zenum,message:string,location:string,_write_ts:time}
type rfb={_path:string,ts:time,uid:string,id:conn_id,client_major_version:string,client_minor_version:string,server_major_version:string,server_minor_version:string,authentication_method:string,auth:bool,share_flag:bool,desktop_name:string,width:uint64,height:uint64,_write_ts:time}
type signatures={_path:string,ts:time,uid:string,src_addr:ip,src_port:port,dst_addr:ip,dst_port:port,note:zenum,sig_id:string,event_msg:string,sub_msg:string,sig_count:uint64,host_count:uint64,_write_ts:time}
type sip={_path:string,ts:time,uid:string,id:conn_id,trans_depth:uint64,method:string,uri:string,date:string,request_from:string,request_to:string,response_from:string,response_to:string,reply_to:string,call_id:string,seq:string,subject:string,request_path:[string],response_path:[string],user_agent:string,status_code:uint64,status_msg:string,warning:string,request_body_len:uint64,response_body_len:uint64,content_type:string,_write_ts:time}
type smb_files={_path:string,ts:time,uid:string,id:conn_id,fuid:string,action:zenum,path:string,name:string,size:uint64,prev_name:string,times:{modified:time,accessed:time,created:time,changed:time},_write_ts:time}
type smb_mapping={_path:string,ts:time,uid:string,id:conn_id,path:string,service:string,native_file_system:string,share_type:string,_write_ts:time}
type smtp={_path:string,ts:time,uid:string,id:conn_id,trans_depth:uint64,helo:string,mailfrom:string,rcptto:|[string]|,date:string,from:string,to:|[string]|,cc:|[string]|,reply_to:string,msg_id:string,in_reply_to:string,subject:string,x_originating_ip:ip,first_received:string,second_received:string,last_reply:string,path:[ip],user_agent:string,tls:bool,fuids:[string],is_webmail:bool,_write_ts:time}
type snmp={_path:string,ts:time,uid:string,id:conn_id,duration:duration,version:string,community:string,get_requests:uint64,get_bulk_requests:uint64,get_responses:uint64,set_requests:uint64,display_string:string,up_since:time,_write_ts:time}
type socks={_path:string,ts:time,uid:string,id:conn_id,version:uint64,user:string,password:string,status:string,request:{host:ip,name:string},request_p:port,bound:{host:ip,name:string},bound_p:port,_write_ts:time}
type software={_path:string,ts:time,host:ip,host_p:port,software_type:zenum,name:string,version:{major:uint64,minor:uint64,minor2:uint64,minor3:uint64,addl:string},unparsed_version:string,_write_ts:time}
type ssh={_path:string,ts:time,uid:string,id:conn_id,version:uint64,auth_success:bool,auth_attempts:uint64,direction:zenum,client:string,server:string,cipher_alg:string,mac_alg:string,compression_alg:string,kex_alg:string,host_key_alg:string,host_key:string,remote_location:{country_code:string,region:string,city:string,latitude:float64,longitude:float64},_write_ts:time}
type ssl={_path:string,ts:time,uid:string,id:conn_id,version:string,cipher:string,curve:string,server_name:string,resumed:bool,last_alert:string,next_protocol:string,established:bool,cert_chain_fuids:[string],client_cert_chain_fuids:[string],subject:string,issuer:string,client_subject:string,client_issuer:string,validation_status:string,_write_ts:time}
type stats={_path:string,ts:time,peer:string,mem:uint64,pkts_proc:uint64,bytes_recv:uint64,pkts_dropped:uint64,pkts_link:uint64,pkt_lag:duration,events_proc:uint64,events_queued:uint64,active_tcp_conns:uint64,active_udp_conns:uint64,active_icmp_conns:uint64,tcp_conns:uint64,udp_conns:uint64,icmp_conns:uint64,timers:uint64,active_timers:uint64,files:uint64,active_files:uint64,dns_requests:uint64,active_dns_requests:uint64,reassem_tcp_size:uint64,reassem_file_size:uint64,reassem_frag_size:uint64,reassem_unknown_size:uint64,_write_ts:time}
type syslog={_path:string,ts:time,uid:string,id:conn_id,proto:zenum,facility:string,severity:string,message:string,_write_ts:time}
type tunnel={_path:string,ts:time,uid:string,id:conn_id,tunnel_type:zenum,action:zenum,_write_ts:time}
type weird={_path:string,ts:time,uid:string,id:conn_id,name:string,addl:string,notice:bool,peer:string,source:string,_write_ts:time}
type x509={_path:string,ts:time,id:string,certificate:{version:uint64,serial:string,subject:string,issuer:string,not_valid_before:time,not_valid_after:time,key_alg:string,sig_alg:string,key_type:string,key_length:uint64,exponent:string,curve:string},san:{dns:[string],uri:[string],email:[string],ip:[ip]},basic_constraints:{ca:bool,path_len:uint64},_write_ts:time}

// This second block of type definitions represent changes needed to cover
// an out-of-the-box Zeek v4.1.0. In other Zeek revisions, we were accustomed
// to only seeing new fields added, but this represented the first time fields
// have changed, e.g., in SSL logs, "cert_chain_fuids" became "cert_chain_fps".
// Therefore we have wholly separate type definitions for this revision so we
// can cover 100% of the expected fields.

type ssl_4_1_0={_path:string,ts:time,uid:string,id:conn_id,version:string,cipher:string,curve:string,server_name:string,resumed:bool,last_alert:string,next_protocol:string,established:bool,ssl_history:string,cert_chain_fps:[string],client_cert_chain_fps:[string],subject:string,issuer:string,client_subject:string,client_issuer:string,sni_matches_cert:bool,validation_status:string,_write_ts:time}
type x509_4_1_0={_path:string,ts:time,fingerprint:string,certificate:{version:uint64,serial:string,subject:string,issuer:string,not_valid_before:time,not_valid_after:time,key_alg:string,sig_alg:string,key_type:string,key_length:uint64,exponent:string,curve:string},san:{dns:[string],uri:[string],email:[string],ip:[ip]},basic_constraints:{ca:bool,path_len:uint64},host_cert:bool,client_cert:bool,_write_ts:time}

const schemas = |{
  "broker": <broker>,
  "capture_loss": <capture_loss>,
  "cluster": <cluster>,
  "config": <config>,
  "conn": <conn>,
  "dce_rpc": <dce_rpc>,
  "dhcp": <dhcp>,
  "dnp3": <dnp3>,
  "dns": <dns>,
  "dpd": <dpd>,
  "files": <files>,
  "ftp": <ftp>,
  "http": <http>,
  "intel": <intel>,
  "irc": <irc>,
  "kerberos": <kerberos>,
  "known_certs": <known_certs>,
  "known_hosts": <known_hosts>,
  "known_services": <known_services>,
  "loaded_scripts": <loaded_scripts>,
  "modbus": <modbus>,
  "mysql": <mysql>,
  "netcontrol": <netcontrol>,
  "netcontrol_drop": <netcontrol_drop>,
  "netcontrol_shunt": <netcontrol_shunt>,
  "notice": <notice>,
  "notice_alarm": <notice_alarm>,
  "ntlm": <ntlm>,
  "ntp": <ntp>,
  "ocsp": <ocsp>,
  "openflow": <openflow>,
  "packet_filter": <packet_filter>,
  "pe": <pe>,
  "radius": <radius>,
  "rdp": <rdp>,
  "reporter": <reporter>,
  "rfb": <rfb>,
  "signatures": <signatures>,
  "sip": <sip>,
  "smb_files": <smb_files>,
  "smb_mapping": <smb_mapping>,
  "smtp": <smtp>,
  "snmp": <snmp>,
  "socks": <socks>,
  "software": <software>,
  "ssh": <ssh>,
  "ssl": <ssl>,
  "stats": <stats>,
  "syslog": <syslog>,
  "tunnel": <tunnel>,
  "weird": <weird>,
  "x509": <x509>
}|

// We'll check for the presence of fields we know are unique to records that
// changed in Zeek v4.1.0 and shape those with special v4.1.0-specific config.
// For everything else we'll apply the default type definitions.

yield nest_dotted(this) | switch (
  case _path=="ssl" and has(ssl_history) => yield shape(<ssl_4_1_0>)
  case _path=="x509" and has(fingerprint) => yield shape(<x509_4_1_0>)
  default => yield shape(schemas[_path])
)
```

### Leading Type Definitions

The top three lines define types that are referenced further below in the main
portion of the Zed shaper.

```
type port=uint16;
type zenum=string;
type conn_id={orig_h:ip,orig_p:port,resp_h:ip,resp_p:port};
```
The `port` and `zenum` types are described further in the [Zed/Zeek Data Type Compatibility](data-type-compatibility.md)
doc. The `conn_id` type will just save us from having to repeat these fields
individually in the many Zeek record types that contain an embedded `id`
record.

### Default Type Definitions Per Zeek Log `_path`

The bulk of this Zed shaper consists of detailed per-field data type
definitions for each record in the default set of NDJSON logs output by Zeek.
These type definitions reference the types we defined above, such as `port`
and `conn_id`. The syntax for defining primitive and complex types follows the
relevant sections of the [ZSON Format](../../formats/zson.md#2-the-zson-format)
specification.

```
...
type conn={_path:string,ts:time,uid:string,id:conn_id,proto:zenum,service:string,duration:duration,orig_bytes:uint64,resp_bytes:uint64,conn_state:string,local_orig:bool,local_resp:bool,missed_bytes:uint64,history:string,orig_pkts:uint64,orig_ip_bytes:uint64,resp_pkts:uint64,resp_ip_bytes:uint64,tunnel_parents:|[string]|,_write_ts:time};
type dce_rpc={_path:string,ts:time,uid:string,id:conn_id,rtt:duration,named_pipe:string,endpoint:string,operation:string,_write_ts:time};
...
```

> **Note:** See [the role of `_path`](reading-zeek-log-formats.md#the-role-of-_path)
> for important details if you're using Zeek's built-in [ASCII logger](https://docs.zeek.org/en/current/scripts/base/frameworks/logging/writers/ascii.zeek.html)
> to generate NDJSON rather than the [JSON Streaming Logs](https://github.com/corelight/json-streaming-logs) package.

### Version-Specific Type Definitions

The next block of type definitions are exceptions for Zeek v4.1.0 where the
names of fields for certain log types have changed from prior releases.

```
type ssl_4_1_0={_path:string,ts:time,uid:string,id:conn_id,version:string,cipher:string,curve:string,server_name:string,resumed:bool,last_alert:string,next_protocol:string,established:bool,ssl_history:string,cert_chain_fps:[string],client_cert_chain_fps:[string],subject:string,issuer:string,client_subject:string,client_issuer:string,sni_matches_cert:bool,validation_status:string,_write_ts:time};
type x509_4_1_0={_path:string,ts:time,fingerprint:string,certificate:{version:uint64,serial:string,subject:string,issuer:string,not_valid_before:time,not_valid_after:time,key_alg:string,sig_alg:string,key_type:string,key_length:uint64,exponent:string,curve:string},san:{dns:[string],uri:[string],email:[string],ip:[ip]},basic_constraints:{ca:bool,path_len:uint64},host_cert:bool,client_cert:bool,_write_ts:time};
```

### Mapping From `_path` Values to Types

The next section is just simple mapping from the string values typically found
in the Zeek `_path` field to the name of one of the types we defined above.

```
const schemas = |{
  "broker": broker,
  "capture_loss": capture_loss,
  "cluster": cluster,
  "config": config,
  "conn": conn,
  "dce_rpc": dce_rpc,
...
```

### Zed Pipeline

The Zed shaper ends with a pipeline that stitches together everything we've defined
so far.

```
put this := unflatten(this) | switch (
  _path=="ssl" has(ssl_history) => put this := shape(ssl_4_1_0);
  _path=="x509" has(fingerprint) => put this := shape(x509_4_1_0);
  default => put this := shape(schemas[_path]);
)
```

Picking this apart, it transforms reach record as it's being read, in three
steps:

1. `unflatten()` reverses the Zeek NDJSON logger's "flattening" of nested
   records, e.g., how it populates a field named `id.orig_h` rather than
   creating a field `id` with sub-field `orig_h` inside it. Restoring the
   original nesting now gives us the option to reference the record named `id`
   in the Zed language and access the entire 4-tuple of values, but still
   access the individual values using the same dotted syntax like `id.orig_h`
   when needed.

2. The `switch()` detects if fields specific to Zeek v4.1.0 are present for the
   two log types for which the [version-specific type definitions](#version-specific-type-definitions)
   should be applied. For all log lines and types other than these exceptions,
   the [default type definitions](#default-type-definitions-per-zeek-log-_path)
   are applied.

3. Each `shape()` call applies an appropriate type definition based on the
   nature of the incoming record. The logic of `shape()` includes:

   * For any fields referenced in the type definition that aren't present in
     the input record, the field is added with a `null` value. (Note: This
     could be performed separately via the `fill()` function.)

   * The data type of each field in the type definition is applied to the
     field of that name in the input record. (Note: This could be performed
     separately via the `cast()` function.)

   * The fields in the input record are ordered to match the order in which
     they appear in the type definition. (Note: This could be performed
     separately via the `order()` function.)

   Any fields that appear in the input record that are not present in the
   type definition are kept and assigned an inferred data type. If you would
   prefer to have such additional fields dropped (i.e., to maintain strict
   adherence to the shape), append a call to the `crop()` function to the
   Zed pipeline, e.g.:

      ```
      ... | put this := shape(schemas[_path]) | put this := crop(schemas[_path])
      ```

   Open issues [zed/2585](https://github.com/brimdata/zed/issues/2585) and
   [zed/2776](https://github.com/brimdata/zed/issues/2776) both track planned
   future improvements to this part of Zed shapers.

## Invoking the Shaper From `zq`

A shaper is typically invoked via the `-I` option of `zq`.

For example, if we assume this input file `weird.ndjson`

```mdtest-input weird.ndjson
{
  "_path": "weird",
  "_write_ts": "2018-03-24T17:15:20.600843Z",
  "ts": "2018-03-24T17:15:20.600843Z",
  "uid": "C1zOivgBT6dBmknqk",
  "id.orig_h": "10.47.1.152",
  "id.orig_p": 49562,
  "id.resp_h": "23.217.103.245",
  "id.resp_p": 80,
  "name": "TCP_ack_underflow_or_misorder",
  "notice": false,
  "peer": "zeek"
}
```

applying the reference shaper via

```mdtest-command
zq -Z -I shaper.zed weird.ndjson
```

produces

```mdtest-output
{
    _path: "weird",
    ts: 2018-03-24T17:15:20.600843Z,
    uid: "C1zOivgBT6dBmknqk",
    id: {
        orig_h: 10.47.1.152,
        orig_p: 49562 (port=uint16),
        resp_h: 23.217.103.245,
        resp_p: 80 (port)
    } (=conn_id),
    name: "TCP_ack_underflow_or_misorder",
    addl: null (string),
    notice: false,
    peer: "zeek",
    source: null (string),
    _write_ts: 2018-03-24T17:15:20.600843Z
} (=weird)
```

If working in a directory containing many NDJSON logs, the
reference shaper can be applied to all the records they contain and
output them all in a single binary [ZNG](../../formats/zng.md) file as
follows:

```
zq -I shaper.zed *.log > /tmp/all.zng
```

If you wish to apply the shaper and then perform additional
operations on the richly-typed records, the Zed query on the command line
should begin with a `|`, as this appends it to the pipeline at the bottom of
the shaper from the included file.

For example, to count Zeek `conn` records into CIDR-based buckets based on
originating IP address:

```
zq -I shaper.zed -f table '| count() by network_of(id.orig_h) | sort -r' conn.log
```

[zed/2584](https://github.com/brimdata/zed/issues/2584) tracks a planned
improvement for this use of `zq -I`.

If you intend to frequently shape the same NDJSON data, you may want to create
an alias in your
shell to always invoke `zq` with the necessary `-I` flag pointing to the path
of your finalized shaper. [zed/1059](https://github.com/brimdata/zed/issues/1059)
tracks a planned enhancement to persist such settings within Zed itself rather
than relying on external mechanisms such as shell aliases.

## Importing Shaped Data Into Zui

If you wish to browse your shaped data with [Zui](https://zui.brimdata.io/),
the best way to accomplish this at the moment would be to use `zq` to convert
it to ZNG [as shown above](#invoking-the-shaper-from-zq), then drag the ZNG
into Zui as you would any other log. An enhancement [zed/2695](https://github.com/brimdata/zed/issues/2695)
is planned that will soon make it possible to attach your shaper to a
Pool. This will allow you to drag the original NDJSON logs directly into the
Pool in Zui and have the shaping applied as the records are being committed to
the Pool.

## Contact us!

If you're having difficulty, interested in shaping other data sources, or
just have feedback, please join our [public Slack](https://www.brimdata.io/join-slack/)
and speak up or [open an issue](https://github.com/brimdata/zed/issues/new/choose).
Thanks!
