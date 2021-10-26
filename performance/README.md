# Performance

The tables below provide a summary of simple operations and how `zq`
performs at them relative to `zeek-cut` and `jq`. All operations were performed
on a Google Cloud `n1-standard-8` VM (8 vCPUs, 30 GB memory) with the logs
stored on a local SSD. `make perf-compare` was used to generate the results.

As there are many results to sift through, here's a few key summary take-aways:

* If all you care about is cutting field values by column, `zeek-cut` does
still perform the best. (Alas, that's all `zeek-cut` can do. :smiley:)

* The numerous input/output formats in `zq` are helpful for fitting into your
legacy pipelines. However, ZNG performs the best of all `zq`-compatible
formats, due to its binary/optimized nature. If you have logs in a non-ZNG
format and expect to query them many times, a one-time pass through `zq` to
convert them to ZNG format will save you significant time.

* Despite it having some CPU cost, the LZ4 compression that `zq` performs by
default when outputting ZNG is shown to have a negligible user-perceptible
performance impact. With this sample data, the LZ4-compressed ZNG is less than
half the size of the uncompressed ZNG.

* Particularly when working in ZNG format & when simple analytics (counting,
grouping) are in play, `zq` can significantly outperform `jq`. That said, `zq`
does not (yet) include the full set of mathematical/other operations available
in `jq`. If there's glaring functional omissions that are limiting your use of
`zq`, we welcome [contributions](../README.md#contributing).

* For the permutations of `ndjson` input and `zeek` output, the recommended
approach for [shaping Zeek NDJSON](../docs/zeek/Shaping-Zeek-NDJSON.md)
was followed as the input data was being read. In addition to conforming to the
best practices as described in that article, this also avoids a problem
described in [a comment in zed/2123](https://github.com/brimdata/zed/pull/2123#issuecomment-859164320).

# Results

The results below reflect performance as of `zq` commit `e425777`.

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|8.98|16.38|1.01|
|`zq`|`*`|zeek|zng|4.49|6.13|0.47|
|`zq`|`*`|zeek|zng-uncompressed|4.29|5.46|0.40|
|`zq`|`*`|zeek|zson|13.85|22.05|1.13|
|`zq`|`*`|zeek|tzng|8.73|14.90|0.75|
|`zq`|`*`|zeek|ndjson|50.65|67.75|3.19|
|`zq`|`*`|zng|zeek|9.13|12.31|0.56|
|`zq`|`*`|zng|zng|3.62|4.26|0.30|
|`zq`|`*`|zng|zng-uncompressed|3.22|3.77|0.31|
|`zq`|`*`|zng|zson|14.23|17.63|0.50|
|`zq`|`*`|zng|tzng|9.06|11.86|0.38|
|`zq`|`*`|zng|ndjson|48.88|55.18|0.99|
|`zq`|`*`|zng-uncompressed|zeek|9.09|12.39|0.41|
|`zq`|`*`|zng-uncompressed|zng|3.43|4.03|0.25|
|`zq`|`*`|zng-uncompressed|zng-uncompressed|3.15|3.64|0.32|
|`zq`|`*`|zng-uncompressed|zson|13.99|17.28|0.49|
|`zq`|`*`|zng-uncompressed|tzng|8.92|11.58|0.41|
|`zq`|`*`|zng-uncompressed|ndjson|48.25|54.75|0.98|
|`zq`|`*`|zson|zeek|111.25|142.38|5.84|
|`zq`|`*`|zson|zng|109.68|128.47|5.43|
|`zq`|`*`|zson|zng-uncompressed|110.06|129.42|5.22|
|`zq`|`*`|zson|zson|112.72|150.67|6.06|
|`zq`|`*`|zson|tzng|111.88|141.95|5.57|
|`zq`|`*`|zson|ndjson|120.94|210.98|8.11|
|`zq`|`*`|tzng|zeek|11.13|22.14|1.09|
|`zq`|`*`|tzng|zng|8.14|10.40|0.62|
|`zq`|`*`|tzng|zng-uncompressed|8.01|9.61|0.60|
|`zq`|`*`|tzng|zson|14.74|28.06|1.05|
|`zq`|`*`|tzng|tzng|10.83|20.44|1.00|
|`zq`|`*`|tzng|ndjson|51.47|73.61|3.33|
|`zq`|`*`|ndjson|zeek|89.13|131.75|5.45|
|`zq`|`*`|ndjson|zng|69.79|88.50|3.96|
|`zq`|`*`|ndjson|zng-uncompressed|69.76|87.84|4.25|
|`zq`|`*`|ndjson|zson|74.23|114.74|5.01|
|`zq`|`*`|ndjson|tzng|71.68|101.45|4.49|
|`zq`|`*`|ndjson|ndjson|78.31|154.69|6.53|
|`zeek-cut`||zeek|zeek-cut|1.32|1.35|0.16|
|`jq`|`-c '.'`|ndjson|ndjson|38.29|41.88|1.69|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|4.95|7.61|0.52|
|`zq`|`cut ts`|zeek|zng|4.64|5.79|0.49|
|`zq`|`cut ts`|zeek|zng-uncompressed|4.66|5.80|0.45|
|`zq`|`cut ts`|zeek|zson|5.02|7.64|0.55|
|`zq`|`cut ts`|zeek|tzng|4.89|7.33|0.50|
|`zq`|`cut ts`|zeek|ndjson|6.73|11.89|0.79|
|`zq`|`cut ts`|zng|zeek|4.42|5.87|0.33|
|`zq`|`cut ts`|zng|zng|3.59|4.16|0.33|
|`zq`|`cut ts`|zng|zng-uncompressed|3.58|4.09|0.37|
|`zq`|`cut ts`|zng|zson|4.51|6.07|0.27|
|`zq`|`cut ts`|zng|tzng|4.33|5.74|0.31|
|`zq`|`cut ts`|zng|ndjson|5.98|9.10|0.38|
|`zq`|`cut ts`|zng-uncompressed|zeek|4.38|5.73|0.37|
|`zq`|`cut ts`|zng-uncompressed|zng|3.41|3.95|0.29|
|`zq`|`cut ts`|zng-uncompressed|zng-uncompressed|3.37|3.86|0.31|
|`zq`|`cut ts`|zng-uncompressed|zson|4.36|5.72|0.27|
|`zq`|`cut ts`|zng-uncompressed|tzng|4.22|5.58|0.26|
|`zq`|`cut ts`|zng-uncompressed|ndjson|6.03|9.03|0.40|
|`zq`|`cut ts`|zson|zeek|109.36|128.97|5.50|
|`zq`|`cut ts`|zson|zng|108.91|126.37|5.12|
|`zq`|`cut ts`|zson|zng-uncompressed|109.48|128.02|5.00|
|`zq`|`cut ts`|zson|zson|109.29|129.64|4.95|
|`zq`|`cut ts`|zson|tzng|108.79|127.85|5.30|
|`zq`|`cut ts`|zson|ndjson|109.80|134.10|5.60|
|`zq`|`cut ts`|tzng|zeek|8.55|11.71|0.66|
|`zq`|`cut ts`|tzng|zng|8.33|9.87|0.61|
|`zq`|`cut ts`|tzng|zng-uncompressed|8.34|9.87|0.60|
|`zq`|`cut ts`|tzng|zson|8.64|12.00|0.61|
|`zq`|`cut ts`|tzng|tzng|8.57|11.49|0.65|
|`zq`|`cut ts`|tzng|ndjson|9.13|16.23|0.78|
|`zq`|`cut ts`|ndjson|zeek|84.52|110.44|4.94|
|`zq`|`cut ts`|ndjson|zng|68.35|84.43|4.06|
|`zq`|`cut ts`|ndjson|zng-uncompressed|69.44|86.54|4.13|
|`zq`|`cut ts`|ndjson|zson|69.68|88.98|4.04|
|`zq`|`cut ts`|ndjson|tzng|68.88|87.17|4.08|
|`zq`|`cut ts`|ndjson|ndjson|69.69|92.31|4.52|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.29|1.37|0.15|
|`jq`|`-c '. \| { ts }'`|ndjson|ndjson|21.08|24.06|1.00|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|4.04|4.50|0.17|
|`zq`|`count()`|zeek|zng|4.03|4.44|0.19|
|`zq`|`count()`|zeek|zng-uncompressed|4.05|4.50|0.17|
|`zq`|`count()`|zeek|zson|4.07|4.50|0.20|
|`zq`|`count()`|zeek|tzng|4.05|4.53|0.14|
|`zq`|`count()`|zeek|ndjson|4.05|4.54|0.12|
|`zq`|`count()`|zng|zeek|2.84|2.93|0.04|
|`zq`|`count()`|zng|zng|2.83|2.90|0.06|
|`zq`|`count()`|zng|zng-uncompressed|2.84|2.94|0.02|
|`zq`|`count()`|zng|zson|2.83|2.90|0.05|
|`zq`|`count()`|zng|tzng|2.88|2.96|0.04|
|`zq`|`count()`|zng|ndjson|2.82|2.92|0.03|
|`zq`|`count()`|zng-uncompressed|zeek|2.72|2.76|0.04|
|`zq`|`count()`|zng-uncompressed|zng|2.73|2.78|0.03|
|`zq`|`count()`|zng-uncompressed|zng-uncompressed|2.74|2.81|0.02|
|`zq`|`count()`|zng-uncompressed|zson|2.73|2.77|0.04|
|`zq`|`count()`|zng-uncompressed|tzng|2.74|2.80|0.02|
|`zq`|`count()`|zng-uncompressed|ndjson|2.72|2.78|0.03|
|`zq`|`count()`|zson|zeek|107.67|124.32|4.45|
|`zq`|`count()`|zson|zng|107.25|123.63|4.25|
|`zq`|`count()`|zson|zng-uncompressed|108.73|126.19|4.86|
|`zq`|`count()`|zson|zson|108.27|125.57|4.52|
|`zq`|`count()`|zson|tzng|107.89|124.80|4.61|
|`zq`|`count()`|zson|ndjson|108.33|125.27|4.59|
|`zq`|`count()`|tzng|zeek|7.85|8.79|0.21|
|`zq`|`count()`|tzng|zng|7.83|8.71|0.27|
|`zq`|`count()`|tzng|zng-uncompressed|7.89|8.74|0.34|
|`zq`|`count()`|tzng|zson|7.87|8.74|0.30|
|`zq`|`count()`|tzng|tzng|7.84|8.68|0.29|
|`zq`|`count()`|tzng|ndjson|7.83|8.74|0.22|
|`zq`|`count()`|ndjson|zeek|84.29|107.69|4.19|
|`zq`|`count()`|ndjson|zng|68.65|84.42|3.69|
|`zq`|`count()`|ndjson|zng-uncompressed|69.28|85.45|4.12|
|`zq`|`count()`|ndjson|zson|68.54|84.41|3.82|
|`zq`|`count()`|ndjson|tzng|68.35|84.17|3.64|
|`zq`|`count()`|ndjson|ndjson|68.35|84.28|3.43|
|`jq`|`-c -s '. \| length'`|ndjson|ndjson|23.11|23.66|3.62|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|4.32|4.86|0.12|
|`zq`|`count() by id.orig_h`|zeek|zng|4.31|4.79|0.16|
|`zq`|`count() by id.orig_h`|zeek|zng-uncompressed|4.31|4.77|0.20|
|`zq`|`count() by id.orig_h`|zeek|zson|4.31|4.83|0.14|
|`zq`|`count() by id.orig_h`|zeek|tzng|4.32|4.84|0.11|
|`zq`|`count() by id.orig_h`|zeek|ndjson|4.31|4.77|0.20|
|`zq`|`count() by id.orig_h`|zng|zeek|3.16|3.26|0.04|
|`zq`|`count() by id.orig_h`|zng|zng|3.15|3.24|0.05|
|`zq`|`count() by id.orig_h`|zng|zng-uncompressed|3.17|3.29|0.02|
|`zq`|`count() by id.orig_h`|zng|zson|3.16|3.26|0.03|
|`zq`|`count() by id.orig_h`|zng|tzng|3.12|3.18|0.08|
|`zq`|`count() by id.orig_h`|zng|ndjson|3.18|3.28|0.04|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zeek|3.02|3.08|0.03|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng|3.02|3.07|0.04|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng-uncompressed|2.98|3.03|0.05|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zson|3.02|3.05|0.06|
|`zq`|`count() by id.orig_h`|zng-uncompressed|tzng|2.98|3.03|0.05|
|`zq`|`count() by id.orig_h`|zng-uncompressed|ndjson|3.02|3.09|0.02|
|`zq`|`count() by id.orig_h`|zson|zeek|108.57|126.19|4.62|
|`zq`|`count() by id.orig_h`|zson|zng|107.82|124.70|4.50|
|`zq`|`count() by id.orig_h`|zson|zng-uncompressed|109.28|128.10|4.52|
|`zq`|`count() by id.orig_h`|zson|zson|107.96|125.21|4.72|
|`zq`|`count() by id.orig_h`|zson|tzng|108.16|125.87|4.27|
|`zq`|`count() by id.orig_h`|zson|ndjson|108.01|125.28|4.69|
|`zq`|`count() by id.orig_h`|tzng|zeek|8.04|8.97|0.23|
|`zq`|`count() by id.orig_h`|tzng|zng|8.06|8.97|0.20|
|`zq`|`count() by id.orig_h`|tzng|zng-uncompressed|8.08|8.96|0.30|
|`zq`|`count() by id.orig_h`|tzng|zson|8.02|8.90|0.26|
|`zq`|`count() by id.orig_h`|tzng|tzng|8.06|8.90|0.31|
|`zq`|`count() by id.orig_h`|tzng|ndjson|8.02|8.92|0.26|
|`zq`|`count() by id.orig_h`|ndjson|zeek|83.68|107.43|4.24|
|`zq`|`count() by id.orig_h`|ndjson|zng|68.26|84.18|3.64|
|`zq`|`count() by id.orig_h`|ndjson|zng-uncompressed|68.84|85.29|3.73|
|`zq`|`count() by id.orig_h`|ndjson|zson|67.67|83.36|3.64|
|`zq`|`count() by id.orig_h`|ndjson|tzng|67.56|82.95|3.79|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|67.80|83.57|3.68|
|`jq`|`-c -s 'group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}'`|ndjson|ndjson|33.62|34.51|3.13|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zeek|4.29|4.75|0.16|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zng|4.27|4.76|0.12|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zng-uncompressed|4.30|4.74|0.19|
|`zq`|`id.resp_h==52.85.83.116`|zeek|zson|4.31|4.79|0.15|
|`zq`|`id.resp_h==52.85.83.116`|zeek|tzng|4.29|4.72|0.19|
|`zq`|`id.resp_h==52.85.83.116`|zeek|ndjson|4.30|4.76|0.17|
|`zq`|`id.resp_h==52.85.83.116`|zng|zeek|2.93|2.96|0.04|
|`zq`|`id.resp_h==52.85.83.116`|zng|zng|2.93|2.98|0.03|
|`zq`|`id.resp_h==52.85.83.116`|zng|zng-uncompressed|2.92|2.96|0.04|
|`zq`|`id.resp_h==52.85.83.116`|zng|zson|2.92|2.98|0.02|
|`zq`|`id.resp_h==52.85.83.116`|zng|tzng|2.93|2.97|0.04|
|`zq`|`id.resp_h==52.85.83.116`|zng|ndjson|2.91|2.96|0.02|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zeek|2.81|2.81|0.04|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zng|2.81|2.82|0.03|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zng-uncompressed|2.82|2.85|0.01|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|zson|2.80|2.79|0.05|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|tzng|2.81|2.83|0.01|
|`zq`|`id.resp_h==52.85.83.116`|zng-uncompressed|ndjson|2.80|2.80|0.03|
|`zq`|`id.resp_h==52.85.83.116`|zson|zeek|107.80|124.66|4.46|
|`zq`|`id.resp_h==52.85.83.116`|zson|zng|107.86|124.39|4.42|
|`zq`|`id.resp_h==52.85.83.116`|zson|zng-uncompressed|108.58|125.85|4.65|
|`zq`|`id.resp_h==52.85.83.116`|zson|zson|107.95|124.60|4.69|
|`zq`|`id.resp_h==52.85.83.116`|zson|tzng|108.19|124.95|4.60|
|`zq`|`id.resp_h==52.85.83.116`|zson|ndjson|107.91|124.75|4.52|
|`zq`|`id.resp_h==52.85.83.116`|tzng|zeek|7.99|8.83|0.26|
|`zq`|`id.resp_h==52.85.83.116`|tzng|zng|7.95|8.78|0.25|
|`zq`|`id.resp_h==52.85.83.116`|tzng|zng-uncompressed|7.97|8.78|0.29|
|`zq`|`id.resp_h==52.85.83.116`|tzng|zson|7.98|8.80|0.29|
|`zq`|`id.resp_h==52.85.83.116`|tzng|tzng|7.96|8.87|0.21|
|`zq`|`id.resp_h==52.85.83.116`|tzng|ndjson|7.96|8.81|0.25|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zeek|83.78|107.01|4.39|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zng|68.12|84.23|3.52|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zng-uncompressed|68.74|85.10|3.77|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|zson|67.60|83.33|3.62|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|tzng|67.64|83.31|3.71|
|`zq`|`id.resp_h==52.85.83.116`|ndjson|ndjson|67.70|83.33|3.69|
|`jq`|`-c '. \| select(.["id.resp_h"]=="52.85.83.116")'`|ndjson|ndjson|18.64|21.30|1.31|

