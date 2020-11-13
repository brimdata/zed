# Performance

The tables below provide a summary of simple operations and how `zq`
performs at them relative to `zeek-cut` and `jq`. All operations were performed
on a Google Cloud `n1-standard-8` VM (8 vCPUs, 30 GB memory) with the logs
stored on a local SSD. `make perf-compare` was used to generate the results.

As there are many results to sift through, here's a few key summary take-aways:

* If all you care about is cutting field values by column, `zeek-cut` does still perform the best. (Alas, that's all `zeek-cut` can do. :smiley:)

* The numerous input/output formats in `zq` are helpful for fitting into your legacy pipelines. However, ZNG performs the best of all `zq`-compatible formats, due to its binary/optimized nature. If you have logs in a non-ZNG format and expect to query them many times, a one-time pass through `zq` to convert them to ZNG format will save you significant time.

* Despite it having some CPU cost, the LZ4 compression that `zq` performs by default when outputting ZNG is shown to have a negligible user-perceptible performance impact. With this sample data, the LZ4-compressed ZNG is less than half the size of the uncompressed ZNG.

* Particularly when working in ZNG format & when simple analytics (counting, grouping) are in play, `zq` can significantly outperform `jq`. That said, `zq` does not (yet) include the full set of mathematical/other operations available in `jq`. If there's glaring functional omisssions that are limiting your use of `zq`, we welcome [contributions](../README.md#contributing).

# Results

The results below reflect performance as of `zq` release `v0.23.0`.

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|14.08|31.21|0.69|
|`zq`|`*`|zeek|zng|8.84|13.32|0.47|
|`zq`|`*`|zeek|zng-uncompressed|8.95|12.82|0.50|
|`zq`|`*`|zeek|tzng|10.15|23.43|0.31|
|`zq`|`*`|zeek|ndjson|53.83|76.23|1.17|
|`zq`|`*`|zng|zeek|13.18|19.12|0.46|
|`zq`|`*`|zng|zng|3.42|4.60|0.24|
|`zq`|`*`|zng|zng-uncompressed|3.68|4.36|0.34|
|`zq`|`*`|zng|tzng|9.38|12.91|0.43|
|`zq`|`*`|zng|ndjson|52.30|61.79|0.89|
|`zq`|`*`|zng-uncompressed|zeek|12.83|18.73|0.58|
|`zq`|`*`|zng-uncompressed|zng|3.48|4.57|0.29|
|`zq`|`*`|zng-uncompressed|zng-uncompressed|3.69|4.41|0.30|
|`zq`|`*`|zng-uncompressed|tzng|9.14|12.66|0.43|
|`zq`|`*`|zng-uncompressed|ndjson|51.58|61.27|0.95|
|`zq`|`*`|tzng|zeek|12.90|26.30|0.51|
|`zq`|`*`|tzng|zng|7.69|9.49|0.54|
|`zq`|`*`|tzng|zng-uncompressed|7.63|8.90|0.43|
|`zq`|`*`|tzng|tzng|9.04|18.91|0.25|
|`zq`|`*`|tzng|ndjson|52.63|70.91|1.11|
|`zq`|`*`|ndjson|zeek|68.05|103.58|1.82|
|`zq`|`*`|ndjson|zng|64.59|83.04|1.54|
|`zq`|`*`|ndjson|zng-uncompressed|65.58|83.43|1.52|
|`zq`|`*`|ndjson|tzng|65.92|95.82|1.56|
|`zq`|`*`|ndjson|ndjson|71.29|148.68|2.24|
|`zeek-cut`|``|zeek|zeek-cut|1.36|1.23|0.10|
|`jq`|`-c "."`|ndjson|ndjson|37.98|4.60|0.88|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|9.58|15.01|0.58|
|`zq`|`cut ts`|zeek|zng|9.77|13.86|0.53|
|`zq`|`cut ts`|zeek|zng-uncompressed|9.35|13.13|0.55|
|`zq`|`cut ts`|zeek|tzng|9.28|14.39|0.49|
|`zq`|`cut ts`|zeek|ndjson|10.25|19.71|0.59|
|`zq`|`cut ts`|zng|zeek|4.02|6.09|0.28|
|`zq`|`cut ts`|zng|zng|3.92|4.67|0.36|
|`zq`|`cut ts`|zng|zng-uncompressed|4.02|4.72|0.37|
|`zq`|`cut ts`|zng|tzng|3.90|5.68|0.31|
|`zq`|`cut ts`|zng|ndjson|5.51|9.26|0.31|
|`zq`|`cut ts`|zng-uncompressed|zeek|3.87|5.86|0.28|
|`zq`|`cut ts`|zng-uncompressed|zng|3.82|4.46|0.40|
|`zq`|`cut ts`|zng-uncompressed|zng-uncompressed|3.81|4.41|0.33|
|`zq`|`cut ts`|zng-uncompressed|tzng|3.77|5.47|0.28|
|`zq`|`cut ts`|zng-uncompressed|ndjson|5.50|9.19|0.33|
|`zq`|`cut ts`|tzng|zeek|8.33|11.31|0.46|
|`zq`|`cut ts`|tzng|zng|8.11|9.50|0.43|
|`zq`|`cut ts`|tzng|zng-uncompressed|7.88|9.16|0.41|
|`zq`|`cut ts`|tzng|tzng|8.20|11.21|0.40|
|`zq`|`cut ts`|tzng|ndjson|8.24|14.53|0.43|
|`zq`|`cut ts`|ndjson|zeek|64.87|83.81|1.43|
|`zq`|`cut ts`|ndjson|zng|64.74|82.25|1.32|
|`zq`|`cut ts`|ndjson|zng-uncompressed|64.62|81.89|1.48|
|`zq`|`cut ts`|ndjson|tzng|65.87|84.95|1.46|
|`zq`|`cut ts`|ndjson|ndjson|66.58|89.70|1.57|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.32|1.27|0.05|
|`jq`|`-c ". \| { ts }"`|ndjson|ndjson|21.35|3.48|0.56|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|8.64|11.64|0.22|
|`zq`|`count()`|zeek|zng|8.58|11.58|0.20|
|`zq`|`count()`|zeek|zng-uncompressed|8.56|11.56|0.22|
|`zq`|`count()`|zeek|tzng|8.49|11.48|0.18|
|`zq`|`count()`|zeek|ndjson|8.54|11.52|0.19|
|`zq`|`count()`|zng|zeek|3.26|3.40|0.06|
|`zq`|`count()`|zng|zng|3.26|3.40|0.07|
|`zq`|`count()`|zng|zng-uncompressed|3.29|3.43|0.06|
|`zq`|`count()`|zng|tzng|3.37|3.54|0.05|
|`zq`|`count()`|zng|ndjson|3.45|3.61|0.06|
|`zq`|`count()`|zng-uncompressed|zeek|3.37|3.45|0.07|
|`zq`|`count()`|zng-uncompressed|zng|3.34|3.41|0.08|
|`zq`|`count()`|zng-uncompressed|zng-uncompressed|3.25|3.33|0.06|
|`zq`|`count()`|zng-uncompressed|tzng|3.23|3.31|0.07|
|`zq`|`count()`|zng-uncompressed|ndjson|3.21|3.29|0.06|
|`zq`|`count()`|tzng|zeek|7.37|7.93|0.07|
|`zq`|`count()`|tzng|zng|7.38|7.93|0.09|
|`zq`|`count()`|tzng|zng-uncompressed|7.44|8.01|0.07|
|`zq`|`count()`|tzng|tzng|7.45|7.98|0.11|
|`zq`|`count()`|tzng|ndjson|7.44|8.02|0.07|
|`zq`|`count()`|ndjson|zeek|65.86|83.06|0.98|
|`zq`|`count()`|ndjson|zng|65.25|82.11|1.08|
|`zq`|`count()`|ndjson|zng-uncompressed|65.74|82.84|1.12|
|`zq`|`count()`|ndjson|tzng|64.59|81.43|0.99|
|`zq`|`count()`|ndjson|ndjson|65.29|82.26|1.05|
|`jq`|`-c -s ". \| length"`|ndjson|ndjson|21.07|3.62|0.65|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|9.24|12.37|0.29|
|`zq`|`count() by id.orig_h`|zeek|zng|9.28|12.58|0.15|
|`zq`|`count() by id.orig_h`|zeek|zng-uncompressed|9.27|12.49|0.24|
|`zq`|`count() by id.orig_h`|zeek|tzng|9.21|12.36|0.26|
|`zq`|`count() by id.orig_h`|zeek|ndjson|9.14|12.27|0.26|
|`zq`|`count() by id.orig_h`|zng|zeek|3.77|3.94|0.07|
|`zq`|`count() by id.orig_h`|zng|zng|3.78|3.96|0.07|
|`zq`|`count() by id.orig_h`|zng|zng-uncompressed|3.89|4.10|0.05|
|`zq`|`count() by id.orig_h`|zng|tzng|3.90|4.09|0.07|
|`zq`|`count() by id.orig_h`|zng|ndjson|3.89|4.09|0.06|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zeek|3.78|3.91|0.07|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng|3.80|3.93|0.07|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng-uncompressed|3.86|3.98|0.08|
|`zq`|`count() by id.orig_h`|zng-uncompressed|tzng|3.91|4.05|0.06|
|`zq`|`count() by id.orig_h`|zng-uncompressed|ndjson|3.87|4.03|0.04|
|`zq`|`count() by id.orig_h`|tzng|zeek|8.26|8.96|0.07|
|`zq`|`count() by id.orig_h`|tzng|zng|7.97|8.60|0.10|
|`zq`|`count() by id.orig_h`|tzng|zng-uncompressed|7.98|8.60|0.10|
|`zq`|`count() by id.orig_h`|tzng|tzng|7.97|8.64|0.05|
|`zq`|`count() by id.orig_h`|tzng|ndjson|7.93|8.52|0.12|
|`zq`|`count() by id.orig_h`|ndjson|zeek|66.41|84.39|0.98|
|`zq`|`count() by id.orig_h`|ndjson|zng|65.58|83.24|0.96|
|`zq`|`count() by id.orig_h`|ndjson|zng-uncompressed|66.44|84.34|1.01|
|`zq`|`count() by id.orig_h`|ndjson|tzng|65.21|82.75|0.97|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|66.53|84.55|0.93|
|`jq`|`-c -s "group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}"`|ndjson|ndjson|19.96|3.40|0.68|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zeek|8.88|11.91|0.20|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng|8.85|11.82|0.26|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng-uncompressed|8.97|12.05|0.18|
|`zq`|`id.resp_h=52.85.83.116`|zeek|tzng|8.99|11.99|0.26|
|`zq`|`id.resp_h=52.85.83.116`|zeek|ndjson|8.97|12.05|0.18|
|`zq`|`id.resp_h=52.85.83.116`|zng|zeek|3.53|3.61|0.06|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng|3.53|3.63|0.04|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng-uncompressed|3.54|3.62|0.07|
|`zq`|`id.resp_h=52.85.83.116`|zng|tzng|3.43|3.51|0.06|
|`zq`|`id.resp_h=52.85.83.116`|zng|ndjson|3.40|3.50|0.04|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zeek|3.31|3.29|0.08|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zng|3.29|3.31|0.05|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zng-uncompressed|3.30|3.32|0.05|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|tzng|3.30|3.33|0.05|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|ndjson|3.28|3.30|0.05|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zeek|7.60|8.18|0.06|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng|7.66|8.23|0.07|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng-uncompressed|7.64|8.22|0.05|
|`zq`|`id.resp_h=52.85.83.116`|tzng|tzng|7.61|8.15|0.09|
|`zq`|`id.resp_h=52.85.83.116`|tzng|ndjson|7.67|8.23|0.07|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zeek|65.43|82.31|1.05|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng|66.00|83.14|0.99|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng-uncompressed|64.73|81.35|1.14|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|tzng|65.53|82.52|0.99|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|ndjson|64.82|81.59|1.00|
|`jq`|`-c ". \| select(.["id.resp_h"]=="52.85.83.116")"`|ndjson|ndjson|19.32|3.61|0.44|
