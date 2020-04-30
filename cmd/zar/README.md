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
You can clone the repo or copy just this directory into your current
directory using subversion:
```
svn checkout https://github.com/brimsec/zq-sample-data/trunk/zng
```

## ingesting the data

Let's take those logs and ingest them into a directory.   We'll make it
easier to run all the commands by setting an environment variable pointing
to the root of the logs tree.
```
mkdir ./logs
setenv ZAR_ROOT `pwd`/logs
```
Now let's in ingest the data using "zar chop".  We are working on more
sophisticated ways to ingest data, but for now zar chop just chops its input
into chunks of approximately equal size.  Zar chop expects its input to be
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
in the traversal.
```
zar zq "count()" _ > counts.zng
```
This writes the output to counts.zng.  Each run of the zq command on a log
file generates a zng output stream and the streams are all strung together
in sequence to create out the output.  So you can run it this way and send
output to stdout (the default if there is no output file) and pipe it to
a plain zq command that will show the output as a table:
```
zar zq "count()" _ | zq -f table -
```
And you can sum all the counts to get a total:
```
zar zq "count()" _ | zq -f text "sum(count)" -
```
which should equal this
```
zq -f text "count()" zng/*.gz
```

## search for an IP

Let's say you want to search for a particular IP across all the zar logs.
This is easy, just say:
```
zar zq "id.orig_h=10.10.23.2" _ | zq -t -
```
However, it's kind of slow like all the stuff above because every record is read to
search for that IP.

We can speed this up by building an index of whatever we want, in this case IP addresses.
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
(BTW, you might see an invalid zng value in the index as
there's currently a bug causing this.
Will be fixed soon but the indexer ignores it and this doesn't affect the correctness here.)

Now if you run "zar find", it will efficiently look through all the index files
instead of the logs and run much faster...
```
zar find :ip=10.10.23.2
```

### micro-indexes

We call these files "micro indexes" because each index pertains to just one
chunk of log file and represents just one indexing rule.

We're not building a massive, integrated inverted index that
tells you exactly where each event is in the event store.  Our model is to
instead build lots of small indexes for each log chunk and index different things
in the different indexes.

### creating more micro-indexes

You can add and delete micro-indexes whenever you want.  Let's say you later
decide you want searches over the uri field to run fast.  You just run
"zar index" again:
```
zar index uri
```
No need to run a massive re-indexing job if you change the indexing rules
since each micro-index is independent of the other.

And now you can run field matches on `uri`:
```
zar find uri=/file
```

If you have a look, you'll see there are index files now for both type ip
and field uri:
```
zar ls -l
```

### operating directly on micro-indexes

Let's say instead of searching for what log chunk a value is in, we want to
actually pull out the zng records that comprise the index.  The syntax
is kind of clunky and we're working to clean it up but you can say...
```
zar find -o - -x zdx:type:ip 10.47.21.138 | zq -t -
```
and you'll get this...
```
#0:record[key:ip]
0:[10.47.21.138;]
0:[10.47.21.138;]
0:[10.47.21.138;]
0:[10.47.21.138;]
```
Hmm, not very useful.   Clearly that value was present in four of the indexes,
but I don't know anything else.
What if we put other information in the index alongside each key?
Then maybe we can do interesting with that extra info.

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
You can run ls to see the files are indeed there:
```
zar ls groupby.zng
```

Actually, we'd like that to be an index.  So, we should add a "key" field
and make sure the file is sorted by key:
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
(TBD: we should probably use ".zdx" to denote a single file in its zdx form after
we fix things to use single-file form instead of bundle form.)

Now I can see my index files for the custom rule I made
```
zar ls custom.zng
```
I can see what's in it now:
```
zq -f table $ZAR_ROOT/20180324/1521912191.526264.zng.zar/custom.zng | head -5
```
You can see the IPs, counts, byte sums, and _path strings.

### zar find with custom index

And now I can go back to my example from before and use "zar find" on the custom
index:
```
zar find -o - -x custom 10.164.94.120 | zq -t -
```
Now we're talking!  And if I take the results and do a little more math to
aggregate the aggregations, I get this:
```
zar find -o - -x custom 10.164.94.120 | zq -f table "sum(count) as count by _path" -
```
And you get
```
_PATH COUNT
dns   8
dpd   8
rdp   208
conn  136
```
We can compute this aggregation now for any IP in the micro-index
without reading any of the original log files!  Pretty cool.

(TBD: count=sum(count) syntax not working right now, also reducers that don't
find fields insert null, then null input deletes the row on the next aggregation)

## Map-reduce

What's really going on here is map-reduce style computation on your log archives
without having to set up a spark or hadoop cluster and write java map-reduce classes.

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
