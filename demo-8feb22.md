## Demo Feb 8 2022

**TL;DR**

Zed is fast!

### jq

Noah did a bunch of work to speed up jq.

Let's look at `count() by _path`

```
cat query.jq

time cat all.ndjson | jq -n -f query.jq

(19s)

time zq -i json "count() by _path" all.ndjson

(11.3s) = 70% faster

(previously before Noah's commit this was like 60s)
```

> NOTE: we need to say `-i json` which we really need to change because it's
> too easy for our user's to actually get the ZSON decoder via auto-detection
> and conclude zq is really slow.  So we will put json autodetect first since
> it will fail on ZSON unquoted field names.

Of course, if you put your JSON in ZNG... `zq` is WAY faster...

```
time zq -i json -o all.zng all.ndjson

(11.5s) one time

time zq "count() by _path" all.zng

(0.25s) = 75,000% faster (100X faster with native not JSON data types in ZNG file)
```
This is part of the reason...
```
ls -lh  all.ndjson all.zng
```

Let's look at a simple filter... `jq` should be stronger...
```
time cat all.ndjson | jq .id > /dev/null

(20.6s)

time zq -i json -f json "cut id" all.ndjson > /dev/null

(16.5s) 25% faster

To ZNG:

time zq -i json -f json "cut id" all.ndjson > /dev/null

(11.8s) 75% faster

(JSON output path uses Go map for each output value)

All in Zedland:

time zq "cut id" all.zng > /dev/null

(0.62s) 33,000% faster
```

**Key Take Away**

`jq` is both _harder_ AND _slower_

Note: search is probably 1000X+ faster brute force and even more with indexes
(though should compare to ripgrep)

## Analytics


First with jq (try both ways from stackoverflow):
```
cat all.ndjson | jq -n '[inputs | .orig_bytes] | reduce .[] as $num (0; .+$num)'

(16s)

time cat all.ndjson | jq -s '[.[].orig_bytes] | add' -

(21s)

```

Now with zq on JSON:
```
time zq -i json 'sum(orig_bytes)' all.ndjson

(11.2s) - 40% faster
```
And with zq on ZNG:
```
time zq 'sum(orig_bytes)' all.zng

(0.3s) - 5300% faster
```

But isn't ZST supposed to do columnar aggregations fast?  Yes!

Did experiment with pushdown (branch `zst-sum`)

Convert to ZST then run hacked-in sum pushdown:
```
zq -f zst -o all.zst all.zng
time zed dev zst cut -s -k orig_bytes all.zst
```
OOPS, this isn't working right now (WRONG ANSWER)... let's look at duckdb

Should be easy to fix because cut then sum works...
```
zed dev zst cut -k orig_bytes all.zst | zq 'sum(orig_bytes)' -
zed dev zst cut -s -k orig_bytes all.zst
```

---

So, let's look at duckdb instead.  This one works.

`orders.parquet` file from duckdb repo...

Convert to zng and zst:
```
time zq -i parquet -o orders.zng orders.parquet

(6.2s) sheesh

time zq -f zst -o orders.zst orders.zng

(0.5s) that's more like it
```
This is annoying:
```
ls -lh orders.zst orders.parquet
```

Compare zng and zst with duckdb on simple sum aggregation:
```
time zq "sum(o_custkey)" orders.zng

(260ms)

time duckdb junk "SELECT SUM(o_custkey) FROM 'orders.parquet'"

(78ms) ugh, duckdb is 300% faster

time zed dev zst cut -s -k orig_bytes all.zst

(50ms) zst pushdown is 50% faster than duckdb!
```
And the answer was correct...

> NOTE: I wanted to do the parquet experiments also on all.zng, but...

```
zq -f parquet -o all.parquet fuse all.zng
```
Parquet doens't have a union!

We need a `deunion` operator to split unions into different column names,
e.g., `fld_1`, `fld_2`, ...

Something like this:
```
zq -f parquet -o all.parquet "fuse | deunion" all.zng
```

This could be tied to parquet-theme release in new release cadence.
