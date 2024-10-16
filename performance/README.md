# Performance

The tables below provide a summary of simple operations and how `zq`
performs at them relative to `zeek-cut` and `jq`. All operations were performed
on an AWS `t3.2xlarge` VM (8 vCPUs, 32 GB memory, 30 GB gp2 SSD).
`make perf-compare` was used to generate the results.

As there are many results to sift through, here's a few key summary take-aways:

* The numerous input/output formats in `zq` are helpful for fitting into your
legacy pipelines. However, ZNG performs the best of all `zq`-compatible
formats, due to its binary/optimized nature. If you have logs in a non-ZNG
format and expect to query them many times, a one-time pass through `zq` to
convert them to ZNG format will save you significant time.

* Despite it having some CPU cost, the LZ4 compression that `zq` performs by
default when outputting ZNG is shown to have a negligible user-perceptible
performance impact. With this sample data, the LZ4-compressed ZNG is less than
half the size of the uncompressed ZNG.

* Particularly when working in ZNG format and when simple analytics (counting,
grouping) are in play, `zq` can significantly outperform `jq`. That said, `zq`
does not (yet) include the full set of mathematical/other operations available
in `jq`. If there's glaring functional omissions that are limiting your use of
`zq`, we welcome [contributions](../README.md#contributing).

* For the permutations with `ndjson` input the recommended approach for
[shaping Zeek NDJSON](https://zed.brimdata.io/docs/integrations/zeek/shaping-zeek-ndjson)
was followed as the input data was being read. In addition to conforming to the
best practices as described in that article, this also avoids a problem
described in [a comment in super/2123](https://github.com/brimdata/super/pull/2123#issuecomment-859164320).
Separate tests on our VM confirmed the shaping portion of the runs with NDJSON
input consumed approximately 5 seconds out of the total run time on each of
these runs.

# Results

The results below reflect performance as of `zq` commit `20a867d`.

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|11.85|12.39|0.19|
|`zq`|`*`|zeek|zng|4.47|4.47|0.08|
|`zq`|`*`|zeek|zng-uncompressed|3.52|3.52|0.08|
|`zq`|`*`|zeek|zson|24.86|27.01|0.73|
|`zq`|`*`|zeek|ndjson|30.59|31.42|0.53|
|`zq`|`*`|zng|zeek|6.79|8.62|0.16|
|`zq`|`*`|zng|zng|1.38|2.53|0.11|
|`zq`|`*`|zng|zng-uncompressed|1.16|1.45|0.08|
|`zq`|`*`|zng|zson|18.35|20.93|0.53|
|`zq`|`*`|zng|ndjson|23.48|25.41|0.33|
|`zq`|`*`|zng-uncompressed|zeek|6.81|8.63|0.23|
|`zq`|`*`|zng-uncompressed|zng|1.50|2.62|0.08|
|`zq`|`*`|zng-uncompressed|zng-uncompressed|1.55|1.77|0.15|
|`zq`|`*`|zng-uncompressed|zson|20.19|23.25|0.69|
|`zq`|`*`|zng-uncompressed|ndjson|24.66|26.65|0.38|
|`zq`|`*`|zson|zeek|179.55|195.58|4.82|
|`zq`|`*`|zson|zng|177.27|190.88|4.37|
|`zq`|`*`|zson|zng-uncompressed|173.23|187.41|4.28|
|`zq`|`*`|zson|zson|188.89|207.11|5.35|
|`zq`|`*`|zson|ndjson|198.45|215.70|5.17|
|`zq`|`*`|ndjson|zeek|28.55|75.46|4.39|
|`zq`|`*`|ndjson|zng|26.69|61.08|3.51|
|`zq`|`*`|ndjson|zng-uncompressed|26.25|59.42|3.35|
|`zq`|`*`|ndjson|zson|32.27|95.64|5.49|
|`zq`|`*`|ndjson|ndjson|37.48|98.96|4.64|
|`zeek-cut`||zeek|zeek-cut|1.50|1.49|0.22|
|`jq`|`-c '.'`|ndjson|ndjson|48.17|51.37|1.91|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut quiet(ts)`|zeek|zeek|9.79|13.44|1.22|
|`zq`|`cut quiet(ts)`|zeek|zng|8.22|11.48|1.05|
|`zq`|`cut quiet(ts)`|zeek|zng-uncompressed|8.79|12.10|1.43|
|`zq`|`cut quiet(ts)`|zeek|zson|10.12|13.82|1.26|
|`zq`|`cut quiet(ts)`|zeek|ndjson|10.04|13.88|1.26|
|`zq`|`cut quiet(ts)`|zng|zeek|2.08|3.79|0.17|
|`zq`|`cut quiet(ts)`|zng|zng|1.42|2.44|0.20|
|`zq`|`cut quiet(ts)`|zng|zng-uncompressed|1.47|2.62|0.15|
|`zq`|`cut quiet(ts)`|zng|zson|2.22|3.90|0.21|
|`zq`|`cut quiet(ts)`|zng|ndjson|2.47|4.19|0.31|
|`zq`|`cut quiet(ts)`|zng-uncompressed|zeek|2.21|3.94|0.29|
|`zq`|`cut quiet(ts)`|zng-uncompressed|zng|1.62|2.63|0.12|
|`zq`|`cut quiet(ts)`|zng-uncompressed|zng-uncompressed|1.47|2.35|0.13|
|`zq`|`cut quiet(ts)`|zng-uncompressed|zson|2.14|3.85|0.24|
|`zq`|`cut quiet(ts)`|zng-uncompressed|ndjson|2.22|3.86|0.21|
|`zq`|`cut quiet(ts)`|zson|zeek|172.78|191.22|5.76|
|`zq`|`cut quiet(ts)`|zson|zng|171.33|188.36|4.68|
|`zq`|`cut quiet(ts)`|zson|zng-uncompressed|174.57|192.35|4.98|
|`zq`|`cut quiet(ts)`|zson|zson|180.15|198.90|6.01|
|`zq`|`cut quiet(ts)`|zson|ndjson|186.63|206.34|6.55|
|`zq`|`cut quiet(ts)`|ndjson|zeek|32.32|72.35|5.94|
|`zq`|`cut quiet(ts)`|ndjson|zng|31.81|68.47|4.53|
|`zq`|`cut quiet(ts)`|ndjson|zng-uncompressed|31.69|68.69|4.92|
|`zq`|`cut quiet(ts)`|ndjson|zson|32.01|72.90|5.29|
|`zq`|`cut quiet(ts)`|ndjson|ndjson|30.25|69.95|4.70|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.68|1.62|0.29|
|`jq`|`-c '. \| { ts }'`|ndjson|ndjson|25.27|28.21|1.69|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count:=count()`|zeek|zeek|3.35|3.37|0.06|
|`zq`|`count:=count()`|zeek|zng|3.77|3.75|0.09|
|`zq`|`count:=count()`|zeek|zng-uncompressed|4.00|4.01|0.08|
|`zq`|`count:=count()`|zeek|zson|3.90|3.89|0.08|
|`zq`|`count:=count()`|zeek|ndjson|3.97|4.02|0.09|
|`zq`|`count:=count()`|zng|zeek|1.25|1.51|0.10|
|`zq`|`count:=count()`|zng|zng|1.22|1.45|0.08|
|`zq`|`count:=count()`|zng|zng-uncompressed|1.43|1.67|0.09|
|`zq`|`count:=count()`|zng|zson|1.22|1.49|0.08|
|`zq`|`count:=count()`|zng|ndjson|1.11|1.32|0.13|
|`zq`|`count:=count()`|zng-uncompressed|zeek|1.47|1.66|0.06|
|`zq`|`count:=count()`|zng-uncompressed|zng|1.37|1.50|0.09|
|`zq`|`count:=count()`|zng-uncompressed|zng-uncompressed|1.37|1.50|0.09|
|`zq`|`count:=count()`|zng-uncompressed|zson|1.37|1.50|0.08|
|`zq`|`count:=count()`|zng-uncompressed|ndjson|1.40|1.58|0.10|
|`zq`|`count:=count()`|zson|zeek|181.69|196.81|4.94|
|`zq`|`count:=count()`|zson|zng|170.71|185.82|4.69|
|`zq`|`count:=count()`|zson|zng-uncompressed|170.71|185.37|4.13|
|`zq`|`count:=count()`|zson|zson|176.58|191.38|4.99|
|`zq`|`count:=count()`|zson|ndjson|170.36|185.81|4.74|
|`zq`|`count:=count()`|ndjson|zeek|30.11|65.13|4.20|
|`zq`|`count:=count()`|ndjson|zng|29.72|64.47|3.92|
|`zq`|`count:=count()`|ndjson|zng-uncompressed|30.67|66.47|4.60|
|`zq`|`count:=count()`|ndjson|zson|29.12|63.77|4.22|
|`zq`|`count:=count()`|ndjson|ndjson|29.28|64.15|4.17|
|`jq`|`-c -s '. \| length'`|ndjson|ndjson|26.39|27.33|3.95|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by quiet(id.orig_h)`|zeek|zeek|3.52|3.54|0.06|
|`zq`|`count() by quiet(id.orig_h)`|zeek|zng|3.99|3.99|0.09|
|`zq`|`count() by quiet(id.orig_h)`|zeek|zng-uncompressed|3.96|3.98|0.06|
|`zq`|`count() by quiet(id.orig_h)`|zeek|zson|3.96|3.98|0.09|
|`zq`|`count() by quiet(id.orig_h)`|zeek|ndjson|4.32|4.30|0.09|
|`zq`|`count() by quiet(id.orig_h)`|zng|zeek|1.19|1.73|0.10|
|`zq`|`count() by quiet(id.orig_h)`|zng|zng|1.28|1.74|0.08|
|`zq`|`count() by quiet(id.orig_h)`|zng|zng-uncompressed|1.22|1.77|0.08|
|`zq`|`count() by quiet(id.orig_h)`|zng|zson|1.30|1.83|0.08|
|`zq`|`count() by quiet(id.orig_h)`|zng|ndjson|1.13|1.63|0.07|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|zeek|1.31|1.71|0.06|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|zng|1.53|1.89|0.09|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|zng-uncompressed|1.41|1.77|0.06|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|zson|1.27|1.66|0.06|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|ndjson|1.91|2.49|0.11|
|`zq`|`count() by quiet(id.orig_h)`|zson|zeek|171.86|187.47|4.60|
|`zq`|`count() by quiet(id.orig_h)`|zson|zng|169.77|185.68|4.11|
|`zq`|`count() by quiet(id.orig_h)`|zson|zng-uncompressed|173.50|188.96|4.40|
|`zq`|`count() by quiet(id.orig_h)`|zson|zson|168.29|183.41|4.02|
|`zq`|`count() by quiet(id.orig_h)`|zson|ndjson|173.91|189.04|4.47|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|zeek|29.37|65.66|4.32|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|zng|28.87|64.13|4.04|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|zng-uncompressed|29.10|65.06|3.90|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|zson|28.58|63.77|4.12|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|ndjson|29.07|65.38|4.26|
|`jq`|`-c -s 'group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}'`|ndjson|ndjson|41.30|42.09|4.29|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zeek|3.74|3.74|0.08|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zng|3.67|3.71|0.05|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zng-uncompressed|3.65|3.65|0.07|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zson|3.58|3.59|0.07|
|`zq`|`id.resp_h==52.85.83.116`|zeek|ndjson|3.82|3.83|0.07|
|`zq`|`id.resp_h==52.85.83.116`|zng|zeek|1.30|1.61|0.08|
|`zq`|`id.resp_h==52.85.83.116`|zng|zng|1.17|1.47|0.07|
|`zq`|`id.resp_h==52.85.83.116`|zng|zng-uncompressed|1.16|1.48|0.08|
|`zq`|`id.resp_h==52.85.83.116`|zng|zson|1.31|1.67|0.05|
|`zq`|`id.resp_h==52.85.83.116`|zng|ndjson|1.29|1.61|0.16|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zeek|1.45|1.68|0.12|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zng|1.45|1.67|0.12|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zng-uncompressed|1.44|1.68|0.06|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zson|1.46|1.69|0.08|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|ndjson|1.51|1.67|0.16|
|`zq`|`id.resp_h==52.85.83.116`|zson|zeek|171.78|187.50|4.21|
|`zq`|`id.resp_h==52.85.83.116`|zson|zng|182.02|198.56|5.45|
|`zq`|`id.resp_h==52.85.83.116`|zson|zng-uncompressed|187.08|202.91|5.71|
|`zq`|`id.resp_h==52.85.83.116`|zson|zson|184.96|200.95|5.05|
|`zq`|`id.resp_h==52.85.83.116`|zson|ndjson|183.24|198.73|5.64|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zeek|30.52|66.89|4.43|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zng|28.92|64.00|4.24|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zng-uncompressed|28.46|63.45|3.86|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zson|27.61|62.06|3.83|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|ndjson|28.29|63.42|4.16|
|`jq`|`-c '. \| select(.["id.resp_h"]=="52.85.83.116")'`|ndjson|ndjson|22.19|25.56|1.44|
