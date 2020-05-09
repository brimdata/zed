# zar overview

> DISCLAIMER: ZAR IS A CURRENTLY A PROTOTYPE UNDER DEVELOPMENT AND IS
> CHANGING QUICKLY.  PLEASE EXPERIMENT WITH IT AND GIVE US FEEDBACK BUT
> IT'S NOT QUITE READY FOR PRODUCTION USE. THE SYNTAX/OPTIONS/OUTPUT ETC
> ARE ALL SUBJECT TO CHANGE.

This is a sketch of an early prototype of zar and related tools for
indexing and search log archives.

We'll use the test data here:
```
https://github.com/brimsec/zq-sample-data/tree/master/zng
```
You can copy the just zng data directory needed for this demo
into your current directory using subversion:
```
svn checkout https://github.com/brimsec/zq-sample-data/trunk/zng
```
Or, you can clone the whole data repo using git and symlink the zng dir:
```
git clone --depth=1 https://github.com/brimsec/zq-sample-data.git
ln -s zq-sample-data/zng
```

## ingesting the data

Let's take those logs and ingest them into a directory.   We'll make it
easier to run all the commands by setting an environment variable pointing
to the root of the logs tree.
```
mkdir ./logs
set ZAR_ROOT=`pwd`/logs
```

Now, let's ingest the data using "zar chop".  We are working on more
sophisticated ways to ingest data (e.g., by the standard time partitioning
techniques of year/month/day/hour etc), but for now zar chop just chops
its input into chunks of approximately equal size.

Zar chop expects its input to be
in the zng format so we'll use zq to take all the zng logs, gunzip them,
and feed them to chop, which here expects its data on stdin.  We'll chop them
into chunks of 25MB, which is very small, but in this example the data set is
fairly small (175MB) and you can always try it out on larger data sets:
```
zq zng/*.gz | zar chop -s 25 -
```

## initializing the archive

You can list the contents of an archive with zar ls...
```
zar ls
```
Hmm, it doesn't show anything yet because we first have to turn the ingested
data into an archive by creating the zar directories:
```
zar mkdirs ./logs
```
Try "zar ls" now and you can see the zar directories.  This is where zar puts
lots of interesting data associated with each ingested log chunk.
```
zar ls
```

## counting is our "hello world"

Now that it's set up, you can do stuff with the archive.  Maybe the simplest thing
is to count up all the events across the archive.  Since the log chunks
are spread all over the archive, we need a way to run "zq" over the
different pieces and aggregate the result.

The zq subcommand of zar lets you do this.  Here's how you run zq
on each log in the archive.  The "_" refers to the current log file
in the traversal:
```
zar zq "count()" _ > counts.zng
```
This invocation of zar traverses the archive, applies the zql "count()" operator
over each log file, and writes the output as a stream of zng data where the
sub-streams are simply concatenated together.
By default, the output is sent to stdout, which means you can
simply pipe the resulting stream to
a vanilla zq command that will show the output as a table:
```
zar zq "count()" _ | zq -f table -
```
which, for example, results in:
```
COUNT
222617
223815
218575
211968
230343
225666
129094
```
Likewise, you could take the stream of event counts and sum
them to get a total:
```
zar zq "count()" _ | zq -f text "sum(count)" -
```
which should have the same result as
```
zq -f text "count()" zng/*.gz
```
or...
```
1462078
```

## search for an IP

Now let's say you want to search for a particular IP across all the zar logs.
This is easy. You just say:
```
zar zq "id.orig_h=10.10.23.2" _ | zq -t -
```
which gives this somewhat cryptic result in text zng format:
```
#zenum=string
#0:record[_path:string,ts:time,uid:bstring,id:record[orig_h:ip,orig_p:port,resp_h:ip,resp_p:port],proto:zenum,service:bstring,duration:duration,orig_bytes:uint64,resp_bytes:uint64,conn_state:bstring,local_orig:bool,local_resp:bool,missed_bytes:uint64,history:bstring,orig_pkts:uint64,orig_ip_bytes:uint64,resp_pkts:uint64,resp_ip_bytes:uint64,tunnel_parents:set[bstring]]
0:[conn;1521911721.307472;C4NuQHXpLAuXjndmi;[10.10.23.2;11;10.0.0.111;0;]icmp;-;1260.819589;23184;0;OTH;-;-;0;-;828;46368;0;0;-;]
```
(If you want to learn more about this format, check out the
[ZNG spec](https://github.com/brimsec/zq/blob/master/zng/docs/spec.md).)

You might have noticed that this is kind of slow --- like all the counting above ---
because every record is read to search for that IP.

We can speed this up by building an index.  Some people think building indexes
is a waste of time, but we think they can be really helpful if you're smart
about how you build them.

Zar lets you pretty much build any
sort of index you'd like and you can even embed whatever custom zql analytics
you would like in a search index.  But for now, let's look at just IP addresses.

The "zar index" command makes it easy to index any field or any zng type.
e.g., to index every value that is of type IP, we simply say
```
zar index :ip
```
For each zar log, this command will find every field of type IP in every log record
and add a key for that field's value to log's index file.

Hmm that was interesting.  If you type
```
zar ls -l
```
You will see all the indexes left behind. They are just zng files.
If you want to see one, just look at it with zq, e.g.
```
zq -t $ZAR_ROOT/20180324/1521912191.526264.zng.zar/zdx:type:ip.zng
```
Now if you run "zar find", it will efficiently look through all the index files
instead of the logs and run much faster...
```
zar find :ip=10.10.23.2
```
In the output here, you'll see this IP exists in exactly one log file:
```
/path/to/ZAR_ROOT/20180324/1521912868.861247.zng
```

## micro-indexes

We call these zng files "micro indexes" because each index pertains to just one
chunk of log file and represents just one indexing rule.  If you're curious about
what's in the index, it's just a sorted list of keyed records along with some
additional zng streams that comprise a constant b-tree index into the sorted list.
But the cool thing here is that everything is just a zng stream.

Instead of building a massive, inverted index with glorious roaring
bitmaps that tell you exactly where each event is in the event store, our model
is to instead build lots of small indexes for each log chunk and index different
things in the different indexes.

## creating more micro-indexes

The beauty of this approach is that you can add and delete micro-indexes
whenever you want.  No need to suffer the fate of a massive reindexing
job when you have a new idea about what to index.

So, let's say you later decide you want searches over the "uri" field to run fast.
You just run "zar index" again but with different parameters:
```
zar index uri
```
And now you can run field matches on `uri`:
```
zar find uri=/file
```
and you'll get
```
/path/to/ZAR_ROOT/20180324/1521911720.600725.zng
/path/to/ZAR_ROOT/20180324/1521912191.526264.zng
```
If you have a look, you'll see there are index files now for both type ip
and field uri:
```
zar ls -l
```

## operating directly on micro-indexes

Let's say instead of searching for what log chunk a value is in, we want to
actually pull out the zng records that comprise the index.  This turns out
to be really powerful in general, but to give you a taste here, you can say...
```
zar find -z -x zdx:type:ip 10.47.21.138 | zq -t -
```
where `-z` says to produce zng output instead of a path listing,
and you'll get this...
```
#zfile=string
#0:record[key:ip,_log:zfile]
0:[10.47.21.138;/path/to/ZAR_ROOT/20180324/1521911720.600725.zng;]
0:[10.47.21.138;/path/to/ZAR_ROOT/20180324/1521911867.742821.zng;]
0:[10.47.21.138;/path/to/ZAR_ROOT/20180324/1521912191.526264.zng;]
0:[10.47.21.138;/path/to/ZAR_ROOT/20180324/1521912390.147127.zng;]
```
The find command adds a column called "_log" (which can be disabled
or customized to a different field name) so you can see where the
search hits came from even when they are combined into a zng stream.
The type of the path field is a "zng alias" --- a sort of logical type ---
where a client can infer the type "zfile" refers to a zng data file.

But, what if we wanted to put even more information in the index
alongside each key?  If we could, it seems we could do arbitrarily
interesting things with this...

## custom indexes

Since everything is a zng file, you can create whatever values you want to
go along with your index keys using zql queries.  Why don't we go back to counting?

Let's create an index keyed on the field id.orig_h and for each unique value of
this key, we'll compute the number of times that value appeared for each zeek
log type.  To do this, we'll run "zar zq" in a way that leaves
these results behind in each zar directory:
```
zar zq -o groupby.zng "count() by _path, id.orig_h" _
```
In this case, the `-o` argument to `zar zq` tells it to leave the results
attached to the log file, in the zar directory associated with that log.
You can run ls to see the files are indeed there:
```
zar ls groupby.zng
```
Actually, instead of just this file hanging around, we'd like to turn it into
an index that `zar find` can make sense out of.  The simplest way to do this
is to add a "key" field and make sure the file is sorted by key:
```
zar zq -o keys.zng "put key=id.orig_h | cut -c id | sort key" groupby.zng
```
(ignore "value is unset" messages... we need to fix this)

Run ls again and you'll see everything is there
```
zar ls -l
```
Since we made the key files, we don't need the old files anymore so we can
delete them
```
zar rm groupby.zng
```

Now, we can convert the sorted-key zng file into an index that "zar find" can
use by running "zar zdx" (the index form of a zng file is called a zdx file).
The -o option provides the prefix of the zdx file.  Let's just called it "custom".
```
zar zdx -o custom keys.zng
```
Now I can see my index files for the custom rule I made
```
zar ls custom.zng
```
I can see what's in it now:
```
zq -f table $ZAR_ROOT/20180324/1521912191.526264.zng.zar/custom.zng | head -10
```
Along with a header describing the zdx layout,
you can see the IPs, counts, and _path strings.

### zar find with custom index

And now I can go back to my example from before and use "zar find" on the custom
index:
```
zar find -z -x custom 10.164.94.120 | zq -t -
```
Now we're talking!  And if youo take the results and do a little more math to
aggregate the aggregations, like this:
```
zar find -z -x custom 10.164.94.120 | zq -f table "sum(count) as count by _path" -
```
You'll get
```
_PATH       COUNT
dns         8
dpd         24
ftp         93
rdp         4116
rfb         3
ssh         1
ssl         9538
conn        26726
http        13485
ntlm        80
smtp        1178
weird       316
notice      35
dce_rpc     2
smb_files   1
smb_mapping 65
```
We can compute this aggregation now for any IP in the micro-index
without reading any of the original log files!  You'll get the same
output from this...
```
zq "id.orig_h=10.164.94.120" ../zng/*.gz | zq -f table "count() by _path" -
```
But using zar with the custom indexes is MUCH faster.  Pretty cool.

## Map-reduce

What's really going on here is map-reduce style computation on your log archives
without having to set up a spark cluster and write java map-reduce classes.

The classic map-reduce example is word count.  Let's do this example with
the uri field in http logs.  First, we map each record that has a uri
to a new record with that uri and a count of 1:
```
zar zq -o words.zng "uri != null | cut uri | put count=1" _
```
again you can look at one of the files...
```
zq -t $ZAR_ROOT/20180324/1521912008.698329.zng.zar/words.zng
```
Now we reduce by aggregating the uri and summing the counts:
```
zar zq -o wordcounts.zng "sum(count) by uri | put count=sum | cut uri,count" words.zng
```
If we were dealing with a huge archive, we could do an approximation by talking
the top 1000 in each zar directory then we could aggregate with another zq
command at the top-level:
```
zar zq "sort -r count | head 1000" wordcounts.zng | zq -f table "sum(count) by uri | sort -r sum | head 10" -
```
and you get the top-ten URIs...
```
URI                     SUM
/wordpress/wp-login.php 6516
/                       5848
/api/get/3/6            4677
/api/get/1/2            4645
/api/get/4/7            4639
/api/get/2/3            4638
/api/get/4/8            4638
/api/get/1/1            4636
/api/get/6/12           4634
/api/get/9/18           4627
```
Pretty cool!

## pipes

We love pipes in the zq project. Make a test file:

```
zq "head 10000" zng/* > pipes.zng
```

You can use pipes in zql expressions like you've seen above:

```
zq -f table "orig_bytes > 100 | count() by id.resp_p | sort -r" pipes.zng
```

Or you can pipe the output of one zq to another...
```
zq "orig_bytes > 100 | count() by id.resp_p" pipes.zng | zq -f table "sort -r" -
```
We were careful to make the output of zq just a stream of zng records.
So whether you are piping within a zql query, or between zq commands, or
between zar and zq, or over the network (ssh zq...), it's all the same.
```
zq "orig_bytes > 100" pipes.zng | zq "count() by id.resp_p" - | zq -f table "sort -r" -
```
In fact, files are self-contained zng streams, so you can just cat them together
and you still end up with a valid zng stream
```
cat pipes.zng pipes.zng > pipes2.zng
zq -f text "count()" pipes.zng
zq -f text "count()" pipes2.zng
```

## cleanup

To clean out all the files you've created in the zar directories and
start over, just run
```
zar rmdirs $ZAR_ROOT
zar mkdirs $ZAR_ROOT
```
This will leave all the ingested log files in place and just clear out
the zar directories tied to log files.
