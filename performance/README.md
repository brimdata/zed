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

The results below reflect performance as of `zq` commit `806aadb`.

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|11.48|27.65|1.09|
|`zq`|`*`|zeek|zng|3.70|8.22|0.52|
|`zq`|`*`|zeek|zng-uncompressed|3.64|7.58|0.52|
|`zq`|`*`|zeek|zson|10.64|22.78|1.10|
|`zq`|`*`|zeek|tzng|7.91|17.19|0.84|
|`zq`|`*`|zeek|ndjson|45.67|72.39|2.67|
|`zq`|`*`|zng|zeek|10.67|17.18|0.63|
|`zq`|`*`|zng|zng|2.47|4.41|0.23|
|`zq`|`*`|zng|zng-uncompressed|2.33|3.86|0.23|
|`zq`|`*`|zng|zson|10.22|15.35|0.57|
|`zq`|`*`|zng|tzng|7.86|11.83|0.51|
|`zq`|`*`|zng|ndjson|42.63|52.76|0.83|
|`zq`|`*`|zng-uncompressed|zeek|14.91|38.28|2.22|
|`zq`|`*`|zng-uncompressed|zng|11.42|22.86|1.87|
|`zq`|`*`|zng-uncompressed|zng-uncompressed|11.27|22.16|1.86|
|`zq`|`*`|zng-uncompressed|zson|13.50|36.48|2.10|
|`zq`|`*`|zng-uncompressed|tzng|12.30|32.52|2.07|
|`zq`|`*`|zng-uncompressed|ndjson|45.71|74.80|2.35|
|`zq`|`*`|zson|zeek|43.58|77.77|2.28|
|`zq`|`*`|zson|zng|41.72|62.76|1.13|
|`zq`|`*`|zson|zng-uncompressed|42.67|63.49|1.93|
|`zq`|`*`|zson|zson|43.32|76.98|1.78|
|`zq`|`*`|zson|tzng|42.16|71.08|1.13|
|`zq`|`*`|zson|ndjson|52.22|112.18|2.24|
|`zq`|`*`|tzng|zeek|11.77|30.00|1.28|
|`zq`|`*`|tzng|zng|4.84|10.27|0.55|
|`zq`|`*`|tzng|zng-uncompressed|4.73|9.48|0.58|
|`zq`|`*`|tzng|zson|10.86|25.21|0.94|
|`zq`|`*`|tzng|tzng|8.30|20.00|0.90|
|`zq`|`*`|tzng|ndjson|45.86|74.60|2.68|
|`zq`|`*`|ndjson|zeek|58.54|115.71|4.12|
|`zq`|`*`|ndjson|zng|55.58|93.51|3.33|
|`zq`|`*`|ndjson|zng-uncompressed|55.55|93.26|3.38|
|`zq`|`*`|ndjson|zson|58.07|113.03|4.08|
|`zq`|`*`|ndjson|tzng|57.06|106.43|3.61|
|`zq`|`*`|ndjson|ndjson|63.20|161.39|5.08|
|`zeek-cut`||zeek|zeek-cut|1.11|1.14|0.14|
|`jq`|`-c '.'`|ndjson|ndjson|32.24|35.22|1.54|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|4.08|10.29|0.59|
|`zq`|`cut ts`|zeek|zng|3.95|8.51|0.64|
|`zq`|`cut ts`|zeek|zng-uncompressed|3.95|8.55|0.57|
|`zq`|`cut ts`|zeek|zson|4.09|10.56|0.49|
|`zq`|`cut ts`|zeek|tzng|4.10|10.12|0.56|
|`zq`|`cut ts`|zeek|ndjson|5.20|15.36|0.70|
|`zq`|`cut ts`|zng|zeek|2.90|6.17|0.24|
|`zq`|`cut ts`|zng|zng|2.67|4.46|0.25|
|`zq`|`cut ts`|zng|zng-uncompressed|2.66|4.42|0.23|
|`zq`|`cut ts`|zng|zson|3.06|6.52|0.27|
|`zq`|`cut ts`|zng|tzng|2.84|5.84|0.23|
|`zq`|`cut ts`|zng|ndjson|4.82|8.98|0.36|
|`zq`|`cut ts`|zng-uncompressed|zeek|11.91|24.43|1.78|
|`zq`|`cut ts`|zng-uncompressed|zng|11.79|22.80|1.86|
|`zq`|`cut ts`|zng-uncompressed|zng-uncompressed|11.81|22.76|1.93|
|`zq`|`cut ts`|zng-uncompressed|zson|12.16|24.94|2.01|
|`zq`|`cut ts`|zng-uncompressed|tzng|11.99|24.12|1.99|
|`zq`|`cut ts`|zng-uncompressed|ndjson|12.21|28.18|2.07|
|`zq`|`cut ts`|zson|zeek|42.15|64.35|1.75|
|`zq`|`cut ts`|zson|zng|41.81|62.26|1.80|
|`zq`|`cut ts`|zson|zng-uncompressed|41.97|62.61|1.69|
|`zq`|`cut ts`|zson|zson|42.01|64.50|1.75|
|`zq`|`cut ts`|zson|tzng|41.52|63.33|1.10|
|`zq`|`cut ts`|zson|ndjson|41.89|66.80|1.13|
|`zq`|`cut ts`|tzng|zeek|5.14|12.35|0.66|
|`zq`|`cut ts`|tzng|zng|4.90|10.38|0.52|
|`zq`|`cut ts`|tzng|zng-uncompressed|4.89|10.33|0.54|
|`zq`|`cut ts`|tzng|zson|5.16|12.48|0.74|
|`zq`|`cut ts`|tzng|tzng|5.14|12.09|0.65|
|`zq`|`cut ts`|tzng|ndjson|5.93|17.35|0.77|
|`zq`|`cut ts`|ndjson|zeek|56.13|96.73|3.53|
|`zq`|`cut ts`|ndjson|zng|55.77|94.17|3.46|
|`zq`|`cut ts`|ndjson|zng-uncompressed|55.93|95.24|3.31|
|`zq`|`cut ts`|ndjson|zson|56.22|96.83|3.90|
|`zq`|`cut ts`|ndjson|tzng|56.29|97.01|3.45|
|`zq`|`cut ts`|ndjson|ndjson|56.94|102.23|3.57|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.10|1.15|0.15|
|`jq`|`-c '. \| { ts }'`|ndjson|ndjson|17.69|19.97|1.29|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|3.65|7.40|0.43|
|`zq`|`count()`|zeek|zng|3.66|7.45|0.38|
|`zq`|`count()`|zeek|zng-uncompressed|3.66|7.40|0.46|
|`zq`|`count()`|zeek|zson|3.65|7.36|0.45|
|`zq`|`count()`|zeek|tzng|3.67|7.44|0.41|
|`zq`|`count()`|zeek|ndjson|3.66|7.41|0.43|
|`zq`|`count()`|zng|zeek|2.39|3.61|0.21|
|`zq`|`count()`|zng|zng|2.43|3.74|0.17|
|`zq`|`count()`|zng|zng-uncompressed|2.41|3.70|0.16|
|`zq`|`count()`|zng|zson|2.40|3.64|0.20|
|`zq`|`count()`|zng|tzng|2.41|3.72|0.17|
|`zq`|`count()`|zng|ndjson|2.43|3.75|0.17|
|`zq`|`count()`|zng-uncompressed|zeek|11.22|21.82|1.79|
|`zq`|`count()`|zng-uncompressed|zng|11.24|21.73|1.91|
|`zq`|`count()`|zng-uncompressed|zng-uncompressed|11.24|21.95|1.74|
|`zq`|`count()`|zng-uncompressed|zson|11.21|21.80|1.84|
|`zq`|`count()`|zng-uncompressed|tzng|11.20|21.71|1.91|
|`zq`|`count()`|zng-uncompressed|ndjson|11.21|21.89|1.71|
|`zq`|`count()`|zson|zeek|41.47|60.86|1.56|
|`zq`|`count()`|zson|zng|40.98|59.97|0.91|
|`zq`|`count()`|zson|zng-uncompressed|41.74|61.65|1.58|
|`zq`|`count()`|zson|zson|40.79|59.64|0.96|
|`zq`|`count()`|zson|tzng|41.65|61.15|1.65|
|`zq`|`count()`|zson|ndjson|41.52|61.15|1.34|
|`zq`|`count()`|tzng|zeek|4.74|9.26|0.45|
|`zq`|`count()`|tzng|zng|4.77|9.28|0.49|
|`zq`|`count()`|tzng|zng-uncompressed|4.75|9.37|0.39|
|`zq`|`count()`|tzng|zson|4.77|9.38|0.41|
|`zq`|`count()`|tzng|tzng|4.77|9.31|0.42|
|`zq`|`count()`|tzng|ndjson|4.77|9.30|0.48|
|`zq`|`count()`|ndjson|zeek|55.69|93.50|3.08|
|`zq`|`count()`|ndjson|zng|55.62|92.26|3.62|
|`zq`|`count()`|ndjson|zng-uncompressed|55.83|93.66|3.12|
|`zq`|`count()`|ndjson|zson|55.67|93.24|3.32|
|`zq`|`count()`|ndjson|tzng|55.76|93.52|3.20|
|`zq`|`count()`|ndjson|ndjson|55.75|93.44|3.28|
|`jq`|`-c -s '. \| length'`|ndjson|ndjson|19.53|20.08|3.03|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|3.71|7.77|0.36|
|`zq`|`count() by id.orig_h`|zeek|zng|3.65|7.55|0.44|
|`zq`|`count() by id.orig_h`|zeek|zng-uncompressed|3.66|7.69|0.33|
|`zq`|`count() by id.orig_h`|zeek|zson|3.66|7.64|0.40|
|`zq`|`count() by id.orig_h`|zeek|tzng|3.65|7.57|0.45|
|`zq`|`count() by id.orig_h`|zeek|ndjson|3.67|7.60|0.45|
|`zq`|`count() by id.orig_h`|zng|zeek|2.55|3.92|0.22|
|`zq`|`count() by id.orig_h`|zng|zng|2.53|3.86|0.23|
|`zq`|`count() by id.orig_h`|zng|zng-uncompressed|2.57|3.96|0.20|
|`zq`|`count() by id.orig_h`|zng|zson|2.57|3.92|0.22|
|`zq`|`count() by id.orig_h`|zng|tzng|2.57|3.98|0.17|
|`zq`|`count() by id.orig_h`|zng|ndjson|2.57|3.92|0.23|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zeek|11.57|22.28|1.99|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng|11.60|22.31|1.94|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng-uncompressed|11.68|22.58|1.85|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zson|11.63|22.37|1.97|
|`zq`|`count() by id.orig_h`|zng-uncompressed|tzng|11.65|22.34|1.97|
|`zq`|`count() by id.orig_h`|zng-uncompressed|ndjson|11.62|22.40|1.93|
|`zq`|`count() by id.orig_h`|zson|zeek|40.79|59.92|1.07|
|`zq`|`count() by id.orig_h`|zson|zng|41.59|61.70|1.63|
|`zq`|`count() by id.orig_h`|zson|zng-uncompressed|41.65|61.80|1.76|
|`zq`|`count() by id.orig_h`|zson|zson|41.58|61.65|1.79|
|`zq`|`count() by id.orig_h`|zson|tzng|41.78|61.90|1.56|
|`zq`|`count() by id.orig_h`|zson|ndjson|41.69|61.80|1.63|
|`zq`|`count() by id.orig_h`|tzng|zeek|4.78|9.55|0.44|
|`zq`|`count() by id.orig_h`|tzng|zng|4.77|9.49|0.49|
|`zq`|`count() by id.orig_h`|tzng|zng-uncompressed|4.76|9.53|0.45|
|`zq`|`count() by id.orig_h`|tzng|zson|4.78|9.54|0.45|
|`zq`|`count() by id.orig_h`|tzng|tzng|4.88|9.82|0.44|
|`zq`|`count() by id.orig_h`|tzng|ndjson|4.85|9.74|0.46|
|`zq`|`count() by id.orig_h`|ndjson|zeek|56.05|94.45|3.13|
|`zq`|`count() by id.orig_h`|ndjson|zng|55.90|93.70|3.01|
|`zq`|`count() by id.orig_h`|ndjson|zng-uncompressed|55.88|94.16|3.26|
|`zq`|`count() by id.orig_h`|ndjson|zson|56.02|94.20|3.38|
|`zq`|`count() by id.orig_h`|ndjson|tzng|56.08|94.18|3.54|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|56.11|94.27|3.50|
|`jq`|`-c -s 'group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}'`|ndjson|ndjson|28.86|29.34|3.04|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zeek|3.63|7.48|0.39|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng|3.65|7.47|0.46|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng-uncompressed|3.67|7.56|0.39|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zson|3.63|7.44|0.45|
|`zq`|`id.resp_h=52.85.83.116`|zeek|tzng|3.63|7.46|0.40|
|`zq`|`id.resp_h=52.85.83.116`|zeek|ndjson|3.67|7.60|0.37|
|`zq`|`id.resp_h=52.85.83.116`|zng|zeek|2.44|3.69|0.17|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng|2.42|3.60|0.20|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng-uncompressed|2.45|3.72|0.16|
|`zq`|`id.resp_h=52.85.83.116`|zng|zson|2.43|3.63|0.19|
|`zq`|`id.resp_h=52.85.83.116`|zng|tzng|2.44|3.71|0.16|
|`zq`|`id.resp_h=52.85.83.116`|zng|ndjson|2.42|3.62|0.17|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zeek|11.05|21.40|1.81|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zng|11.10|21.43|1.76|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zng-uncompressed|11.02|21.44|1.69|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zson|11.08|21.47|1.75|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|tzng|11.10|21.47|1.74|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|ndjson|11.08|21.52|1.68|
|`zq`|`id.resp_h=52.85.83.116`|zson|zeek|41.27|60.62|1.46|
|`zq`|`id.resp_h=52.85.83.116`|zson|zng|41.76|62.01|1.67|
|`zq`|`id.resp_h=52.85.83.116`|zson|zng-uncompressed|41.42|60.90|1.47|
|`zq`|`id.resp_h=52.85.83.116`|zson|zson|42.10|62.44|1.98|
|`zq`|`id.resp_h=52.85.83.116`|zson|tzng|41.65|61.12|1.53|
|`zq`|`id.resp_h=52.85.83.116`|zson|ndjson|41.70|61.69|1.55|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zeek|4.77|9.39|0.50|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng|4.76|9.31|0.51|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng-uncompressed|4.74|9.38|0.45|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zson|4.81|9.43|0.51|
|`zq`|`id.resp_h=52.85.83.116`|tzng|tzng|4.78|9.43|0.49|
|`zq`|`id.resp_h=52.85.83.116`|tzng|ndjson|4.77|9.37|0.50|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zeek|55.73|93.50|3.17|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng|55.62|92.81|3.01|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng-uncompressed|55.62|92.95|3.55|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zson|55.78|93.53|3.26|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|tzng|55.72|93.28|3.36|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|ndjson|55.68|93.32|3.18|
|`jq`|`-c '. \| select(.["id.resp_h"]=="52.85.83.116")'`|ndjson|ndjson|15.80|18.32|0.99|
