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

The results below reflect performance as of `zq` commit `e01700de`.

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|16.49|44.28|1.13|
|`zq`|`*`|zeek|zng|11.50|25.66|0.95|
|`zq`|`*`|zeek|zng-uncompressed|11.61|25.48|0.85|
|`zq`|`*`|zeek|tzng|13.24|37.55|0.85|
|`zq`|`*`|zeek|ndjson|53.80|87.31|1.56|
|`zq`|`*`|zng|zeek|15.48|31.25|0.77|
|`zq`|`*`|zng|zng|10.27|14.07|0.64|
|`zq`|`*`|zng|zng-uncompressed|10.36|13.67|0.59|
|`zq`|`*`|zng|tzng|11.67|23.92|0.55|
|`zq`|`*`|zng|ndjson|52.16|72.71|1.13|
|`zq`|`*`|zng-uncompressed|zeek|23.85|51.58|3.73|
|`zq`|`*`|zng-uncompressed|zng|21.56|32.32|3.23|
|`zq`|`*`|zng-uncompressed|zng-uncompressed|21.44|31.43|3.25|
|`zq`|`*`|zng-uncompressed|tzng|22.00|42.45|3.34|
|`zq`|`*`|zng-uncompressed|ndjson|55.36|94.82|4.03|
|`zq`|`*`|tzng|zeek|15.96|40.65|0.83|
|`zq`|`*`|tzng|zng|10.58|21.25|0.59|
|`zq`|`*`|tzng|zng-uncompressed|10.75|21.00|0.67|
|`zq`|`*`|tzng|tzng|12.15|32.39|0.70|
|`zq`|`*`|tzng|ndjson|53.49|83.33|1.46|
|`zq`|`*`|ndjson|zeek|57.06|145.47|2.15|
|`zq`|`*`|ndjson|zng|54.38|117.43|1.78|
|`zq`|`*`|ndjson|zng-uncompressed|54.75|117.23|1.88|
|`zq`|`*`|ndjson|tzng|55.19|131.88|1.96|
|`zq`|`*`|ndjson|ndjson|65.05|182.80|2.41|
|`zeek-cut`||zeek|zeek-cut|1.35|1.39|0.18|
|`jq`|`-c "."`|ndjson|ndjson|38.19|41.81|2.05|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|11.81|27.20|0.78|
|`zq`|`cut ts`|zeek|zng|12.00|26.29|0.87|
|`zq`|`cut ts`|zeek|zng-uncompressed|11.43|24.74|0.80|
|`zq`|`cut ts`|zeek|tzng|11.77|26.65|0.81|
|`zq`|`cut ts`|zeek|ndjson|12.53|32.39|0.89|
|`zq`|`cut ts`|zng|zeek|10.48|15.23|0.54|
|`zq`|`cut ts`|zng|zng|10.60|13.87|0.50|
|`zq`|`cut ts`|zng|zng-uncompressed|10.30|13.43|0.58|
|`zq`|`cut ts`|zng|tzng|10.20|14.58|0.46|
|`zq`|`cut ts`|zng|ndjson|10.83|19.18|0.68|
|`zq`|`cut ts`|zng-uncompressed|zeek|21.33|32.41|3.46|
|`zq`|`cut ts`|zng-uncompressed|zng|21.13|30.68|3.31|
|`zq`|`cut ts`|zng-uncompressed|zng-uncompressed|21.17|30.41|3.55|
|`zq`|`cut ts`|zng-uncompressed|tzng|21.26|32.22|3.19|
|`zq`|`cut ts`|zng-uncompressed|ndjson|21.68|36.82|3.39|
|`zq`|`cut ts`|tzng|zeek|10.77|22.32|0.72|
|`zq`|`cut ts`|tzng|zng|10.52|20.18|0.67|
|`zq`|`cut ts`|tzng|zng-uncompressed|10.58|20.10|0.72|
|`zq`|`cut ts`|tzng|tzng|10.60|21.55|0.65|
|`zq`|`cut ts`|tzng|ndjson|11.44|27.38|0.66|
|`zq`|`cut ts`|ndjson|zeek|53.96|117.15|2.00|
|`zq`|`cut ts`|ndjson|zng|53.69|115.38|1.81|
|`zq`|`cut ts`|ndjson|zng-uncompressed|53.53|114.77|1.84|
|`zq`|`cut ts`|ndjson|tzng|53.60|116.40|1.75|
|`zq`|`cut ts`|ndjson|ndjson|53.90|121.50|1.90|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.29|1.39|0.14|
|`jq`|`-c ". \| { ts }"`|ndjson|ndjson|20.94|23.66|1.31|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|11.07|23.56|0.56|
|`zq`|`count()`|zeek|zng|10.95|23.34|0.58|
|`zq`|`count()`|zeek|zng-uncompressed|11.40|24.30|0.56|
|`zq`|`count()`|zeek|tzng|11.15|23.60|0.56|
|`zq`|`count()`|zeek|ndjson|10.99|23.34|0.59|
|`zq`|`count()`|zng|zeek|9.75|12.10|0.34|
|`zq`|`count()`|zng|zng|9.97|12.54|0.30|
|`zq`|`count()`|zng|zng-uncompressed|10.04|12.58|0.35|
|`zq`|`count()`|zng|tzng|9.94|12.33|0.32|
|`zq`|`count()`|zng|ndjson|9.88|12.35|0.26|
|`zq`|`count()`|zng-uncompressed|zeek|20.78|29.95|3.19|
|`zq`|`count()`|zng-uncompressed|zng|20.77|29.98|3.13|
|`zq`|`count()`|zng-uncompressed|zng-uncompressed|20.59|29.66|3.10|
|`zq`|`count()`|zng-uncompressed|tzng|20.48|29.25|3.48|
|`zq`|`count()`|zng-uncompressed|ndjson|20.69|29.50|3.53|
|`zq`|`count()`|tzng|zeek|9.92|18.65|0.35|
|`zq`|`count()`|tzng|zng|9.92|18.49|0.41|
|`zq`|`count()`|tzng|zng-uncompressed|9.81|18.36|0.41|
|`zq`|`count()`|tzng|tzng|9.81|18.47|0.37|
|`zq`|`count()`|tzng|ndjson|9.77|18.28|0.40|
|`zq`|`count()`|ndjson|zeek|52.91|112.08|1.58|
|`zq`|`count()`|ndjson|zng|53.06|112.33|1.47|
|`zq`|`count()`|ndjson|zng-uncompressed|53.65|113.55|1.53|
|`zq`|`count()`|ndjson|tzng|53.44|113.01|1.45|
|`zq`|`count()`|ndjson|ndjson|54.24|114.84|1.49|
|`jq`|`-c -s ". \| length"`|ndjson|ndjson|23.26|24.03|3.26|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|11.64|24.74|0.57|
|`zq`|`count() by id.orig_h`|zeek|zng|11.61|24.61|0.62|
|`zq`|`count() by id.orig_h`|zeek|zng-uncompressed|11.48|24.54|0.58|
|`zq`|`count() by id.orig_h`|zeek|tzng|11.59|24.58|0.73|
|`zq`|`count() by id.orig_h`|zeek|ndjson|11.63|24.83|0.58|
|`zq`|`count() by id.orig_h`|zng|zeek|10.63|13.23|0.30|
|`zq`|`count() by id.orig_h`|zng|zng|10.25|12.79|0.23|
|`zq`|`count() by id.orig_h`|zng|zng-uncompressed|10.60|13.30|0.25|
|`zq`|`count() by id.orig_h`|zng|tzng|10.89|13.61|0.21|
|`zq`|`count() by id.orig_h`|zng|ndjson|10.70|13.28|0.33|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zeek|22.09|31.96|3.40|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng|22.37|32.44|3.44|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng-uncompressed|22.05|31.93|3.31|
|`zq`|`count() by id.orig_h`|zng-uncompressed|tzng|22.07|31.80|3.54|
|`zq`|`count() by id.orig_h`|zng-uncompressed|ndjson|22.34|32.49|3.30|
|`zq`|`count() by id.orig_h`|tzng|zeek|10.67|19.87|0.41|
|`zq`|`count() by id.orig_h`|tzng|zng|10.64|19.77|0.41|
|`zq`|`count() by id.orig_h`|tzng|zng-uncompressed|10.44|19.41|0.43|
|`zq`|`count() by id.orig_h`|tzng|tzng|10.53|19.50|0.41|
|`zq`|`count() by id.orig_h`|tzng|ndjson|10.60|19.73|0.42|
|`zq`|`count() by id.orig_h`|ndjson|zeek|55.83|119.51|1.49|
|`zq`|`count() by id.orig_h`|ndjson|zng|53.28|113.99|1.46|
|`zq`|`count() by id.orig_h`|ndjson|zng-uncompressed|53.44|114.38|1.54|
|`zq`|`count() by id.orig_h`|ndjson|tzng|53.45|114.44|1.47|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|53.52|114.33|1.68|
|`jq`|`-c -s "group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}"`|ndjson|ndjson|34.01|34.74|3.35|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zeek|11.00|23.33|0.57|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng|11.02|23.44|0.52|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng-uncompressed|11.10|23.50|0.56|
|`zq`|`id.resp_h=52.85.83.116`|zeek|tzng|11.19|23.64|0.51|
|`zq`|`id.resp_h=52.85.83.116`|zeek|ndjson|11.00|23.36|0.54|
|`zq`|`id.resp_h=52.85.83.116`|zng|zeek|9.94|12.25|0.35|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng|9.92|12.33|0.23|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng-uncompressed|10.20|12.55|0.33|
|`zq`|`id.resp_h=52.85.83.116`|zng|tzng|9.97|12.28|0.32|
|`zq`|`id.resp_h=52.85.83.116`|zng|ndjson|10.02|12.45|0.25|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zeek|20.58|29.50|3.18|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zng|20.71|29.53|3.31|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zng-uncompressed|20.66|29.45|3.35|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|tzng|20.48|29.29|3.25|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|ndjson|20.72|29.72|3.19|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zeek|10.08|18.79|0.42|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng|9.79|18.33|0.36|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng-uncompressed|9.85|18.39|0.37|
|`zq`|`id.resp_h=52.85.83.116`|tzng|tzng|10.12|18.94|0.31|
|`zq`|`id.resp_h=52.85.83.116`|tzng|ndjson|9.79|18.39|0.32|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zeek|52.75|111.75|1.44|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng|53.49|113.23|1.52|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng-uncompressed|53.77|113.99|1.45|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|tzng|54.43|115.00|1.68|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|ndjson|55.38|117.00|1.73|
|`jq`|`-c ". \| select(.["id.resp_h"]=="52.85.83.116")"`|ndjson|ndjson|19.22|21.85|1.44|
