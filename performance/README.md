# Performance

The tables below provide a summary of simple operations and how `zq`
performs at them relative to `zeek-cut` and `jq`. All operations were performed
on an AWS `t2.2xlarge` VM (8 vCPUs, 32 GB memory, 30 GB gp2 SSD).
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
described in [a comment in zed/2123](https://github.com/brimdata/zed/pull/2123#issuecomment-859164320).
Separate tests on our VM confirmed the shaping portion of the runs with NDJSON
input consumed approximately 5 seconds out of the total run time on each of
these runs.

# Results

The results below reflect performance as of `zq` commit `4ffdf3e`.

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|11.92|13.00|0.20|
|`zq`|`*`|zeek|zng|4.09|4.12|0.05|
|`zq`|`*`|zeek|zng-uncompressed|3.32|3.38|0.02|
|`zq`|`*`|zeek|zson|18.61|20.29|0.19|
|`zq`|`*`|zeek|ndjson|28.95|31.66|0.29|
|`zq`|`*`|zng|zeek|8.68|10.55|0.19|
|`zq`|`*`|zng|zng|1.19|2.13|0.04|
|`zq`|`*`|zng|zng-uncompressed|1.15|1.32|0.04|
|`zq`|`*`|zng|zson|15.90|17.98|0.27|
|`zq`|`*`|zng|ndjson|26.68|29.49|0.27|
|`zq`|`*`|zng-uncompressed|zeek|8.89|10.75|0.20|
|`zq`|`*`|zng-uncompressed|zng|1.27|2.13|0.06|
|`zq`|`*`|zng-uncompressed|zng-uncompressed|1.22|1.31|0.04|
|`zq`|`*`|zng-uncompressed|zson|15.89|18.04|0.21|
|`zq`|`*`|zng-uncompressed|ndjson|26.55|29.39|0.23|
|`zq`|`*`|zson|zeek|145.46|157.24|1.03|
|`zq`|`*`|zson|zng|136.84|145.84|0.65|
|`zq`|`*`|zson|zng-uncompressed|136.34|146.21|0.60|
|`zq`|`*`|zson|zson|152.96|165.29|1.07|
|`zq`|`*`|zson|ndjson|168.16|182.40|1.05|
|`zq`|`*`|ndjson|zeek|19.49|46.10|1.53|
|`zq`|`*`|ndjson|zng|18.78|34.54|0.73|
|`zq`|`*`|ndjson|zng-uncompressed|18.35|33.55|0.85|
|`zq`|`*`|ndjson|zson|20.62|55.67|1.79|
|`zq`|`*`|ndjson|ndjson|31.12|68.29|2.13|
|`zeek-cut`||zeek|zeek-cut|1.31|1.34|0.15|
|`jq`|`-c '.'`|ndjson|ndjson|43.09|45.47|1.23|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|5.36|5.62|0.09|
|`zq`|`cut ts`|zeek|zng|4.26|4.41|0.04|
|`zq`|`cut ts`|zeek|zng-uncompressed|4.13|4.31|0.03|
|`zq`|`cut ts`|zeek|zson|5.15|5.43|0.04|
|`zq`|`cut ts`|zeek|ndjson|6.35|6.63|0.10|
|`zq`|`cut ts`|zng|zeek|1.95|3.25|0.09|
|`zq`|`cut ts`|zng|zng|1.20|2.06|0.06|
|`zq`|`cut ts`|zng|zng-uncompressed|1.19|1.93|0.07|
|`zq`|`cut ts`|zng|zson|1.84|3.14|0.09|
|`zq`|`cut ts`|zng|ndjson|2.92|4.27|0.06|
|`zq`|`cut ts`|zng-uncompressed|zeek|1.97|3.28|0.09|
|`zq`|`cut ts`|zng-uncompressed|zng|1.27|2.06|0.06|
|`zq`|`cut ts`|zng-uncompressed|zng-uncompressed|1.26|1.91|0.11|
|`zq`|`cut ts`|zng-uncompressed|zson|1.86|3.17|0.06|
|`zq`|`cut ts`|zng-uncompressed|ndjson|2.92|4.26|0.09|
|`zq`|`cut ts`|zson|zeek|148.62|159.81|0.93|
|`zq`|`cut ts`|zson|zng|146.67|156.38|0.65|
|`zq`|`cut ts`|zson|zng-uncompressed|146.75|157.07|0.76|
|`zq`|`cut ts`|zson|zson|145.47|156.70|0.88|
|`zq`|`cut ts`|zson|ndjson|146.95|158.13|0.96|
|`zq`|`cut ts`|ndjson|zeek|19.10|36.76|1.08|
|`zq`|`cut ts`|ndjson|zng|18.98|34.97|1.00|
|`zq`|`cut ts`|ndjson|zng-uncompressed|19.06|35.37|0.91|
|`zq`|`cut ts`|ndjson|zson|19.58|37.25|1.07|
|`zq`|`cut ts`|ndjson|ndjson|19.75|38.44|1.19|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.32|1.41|0.14|
|`jq`|`-c '. \| { ts }'`|ndjson|ndjson|21.33|23.75|0.92|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count:=count()`|zeek|zeek|3.55|3.60|0.03|
|`zq`|`count:=count()`|zeek|zng|3.54|3.59|0.03|
|`zq`|`count:=count()`|zeek|zng-uncompressed|3.55|3.59|0.05|
|`zq`|`count:=count()`|zeek|zson|3.55|3.59|0.04|
|`zq`|`count:=count()`|zeek|ndjson|3.55|3.59|0.04|
|`zq`|`count:=count()`|zng|zeek|1.18|1.32|0.04|
|`zq`|`count:=count()`|zng|zng|1.18|1.32|0.03|
|`zq`|`count:=count()`|zng|zng-uncompressed|1.18|1.32|0.04|
|`zq`|`count:=count()`|zng|zson|1.18|1.31|0.04|
|`zq`|`count:=count()`|zng|ndjson|1.18|1.30|0.06|
|`zq`|`count:=count()`|zng-uncompressed|zeek|1.26|1.34|0.03|
|`zq`|`count:=count()`|zng-uncompressed|zng|1.25|1.32|0.04|
|`zq`|`count:=count()`|zng-uncompressed|zng-uncompressed|1.26|1.32|0.05|
|`zq`|`count:=count()`|zng-uncompressed|zson|1.26|1.32|0.05|
|`zq`|`count:=count()`|zng-uncompressed|ndjson|1.26|1.33|0.03|
|`zq`|`count:=count()`|zson|zeek|150.67|159.51|0.80|
|`zq`|`count:=count()`|zson|zng|150.22|158.95|0.83|
|`zq`|`count:=count()`|zson|zng-uncompressed|149.28|159.04|0.87|
|`zq`|`count:=count()`|zson|zson|150.54|159.61|0.90|
|`zq`|`count:=count()`|zson|ndjson|149.06|158.27|0.80|
|`zq`|`count:=count()`|ndjson|zeek|18.26|34.03|1.00|
|`zq`|`count:=count()`|ndjson|zng|18.18|33.88|1.01|
|`zq`|`count:=count()`|ndjson|zng-uncompressed|18.22|34.01|0.99|
|`zq`|`count:=count()`|ndjson|zson|18.25|34.08|0.91|
|`zq`|`count:=count()`|ndjson|ndjson|18.00|33.41|0.94|
|`jq`|`-c -s '. \| length'`|ndjson|ndjson|23.31|23.13|3.09|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by quiet(id.orig_h)`|zeek|zeek|3.82|3.94|0.05|
|`zq`|`count() by quiet(id.orig_h)`|zeek|zng|3.80|3.92|0.05|
|`zq`|`count() by quiet(id.orig_h)`|zeek|zng-uncompressed|3.82|3.97|0.02|
|`zq`|`count() by quiet(id.orig_h)`|zeek|zson|3.81|3.95|0.03|
|`zq`|`count() by quiet(id.orig_h)`|zeek|ndjson|3.84|3.98|0.02|
|`zq`|`count() by quiet(id.orig_h)`|zng|zeek|1.17|1.88|0.03|
|`zq`|`count() by quiet(id.orig_h)`|zng|zng|1.17|1.83|0.08|
|`zq`|`count() by quiet(id.orig_h)`|zng|zng-uncompressed|1.17|1.85|0.04|
|`zq`|`count() by quiet(id.orig_h)`|zng|zson|1.17|1.86|0.04|
|`zq`|`count() by quiet(id.orig_h)`|zng|ndjson|1.17|1.85|0.05|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|zeek|1.24|1.85|0.06|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|zng|1.26|1.88|0.05|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|zng-uncompressed|1.26|1.85|0.07|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|zson|1.25|1.84|0.07|
|`zq`|`count() by quiet(id.orig_h)`|zng-uncompressed|ndjson|1.24|1.86|0.06|
|`zq`|`count() by quiet(id.orig_h)`|zson|zeek|136.67|146.23|0.84|
|`zq`|`count() by quiet(id.orig_h)`|zson|zng|136.94|146.37|0.80|
|`zq`|`count() by quiet(id.orig_h)`|zson|zng-uncompressed|137.36|147.12|0.73|
|`zq`|`count() by quiet(id.orig_h)`|zson|zson|143.32|153.18|0.95|
|`zq`|`count() by quiet(id.orig_h)`|zson|ndjson|149.68|159.91|0.80|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|zeek|18.97|35.64|1.00|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|zng|18.88|35.23|0.98|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|zng-uncompressed|18.97|35.43|1.09|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|zson|19.04|35.44|1.09|
|`zq`|`count() by quiet(id.orig_h)`|ndjson|ndjson|19.09|35.44|1.15|
|`jq`|`-c -s 'group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}'`|ndjson|ndjson|34.25|34.65|2.90|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zeek|3.91|4.03|0.03|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zng|3.91|4.02|0.04|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zng-uncompressed|3.73|3.85|0.03|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zson|3.64|3.75|0.03|
|`zq`|`id.resp_h==52.85.83.116`|zeek|ndjson|3.69|3.81|0.04|
|`zq`|`id.resp_h==52.85.83.116`|zng|zeek|1.17|1.66|0.06|
|`zq`|`id.resp_h==52.85.83.116`|zng|zng|1.17|1.68|0.04|
|`zq`|`id.resp_h==52.85.83.116`|zng|zng-uncompressed|1.18|1.65|0.07|
|`zq`|`id.resp_h==52.85.83.116`|zng|zson|1.17|1.70|0.02|
|`zq`|`id.resp_h==52.85.83.116`|zng|ndjson|1.17|1.70|0.02|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zeek|1.25|1.67|0.06|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zng|1.24|1.68|0.04|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zng-uncompressed|1.24|1.67|0.05|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zson|1.24|1.68|0.05|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|ndjson|1.24|1.67|0.06|
|`zq`|`id.resp_h==52.85.83.116`|zson|zeek|146.41|157.41|0.86|
|`zq`|`id.resp_h==52.85.83.116`|zson|zng|138.62|149.02|0.74|
|`zq`|`id.resp_h==52.85.83.116`|zson|zng-uncompressed|135.65|146.10|0.86|
|`zq`|`id.resp_h==52.85.83.116`|zson|zson|132.49|143.00|0.71|
|`zq`|`id.resp_h==52.85.83.116`|zson|ndjson|132.51|142.94|0.79|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zeek|18.22|33.73|0.93|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zng|18.17|33.58|0.86|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zng-uncompressed|18.32|33.90|0.96|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zson|18.23|33.75|0.91|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|ndjson|18.22|33.60|1.01|
|`jq`|`-c '. \| select(.["id.resp_h"]=="52.85.83.116")'`|ndjson|ndjson|17.90|20.19|0.97|

