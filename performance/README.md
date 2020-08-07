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

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|16.65|44.33|1.60|
|`zq`|`*`|zeek|zng|8.81|16.15|0.78|
|`zq`|`*`|zeek|zng-uncompressed|8.74|15.31|0.77|
|`zq`|`*`|zeek|tzng|10.90|28.47|0.85|
|`zq`|`*`|zeek|ndjson|61.02|105.16|3.37|
|`zq`|`*`|zng|zeek|14.30|21.56|0.64|
|`zq`|`*`|zng|zng|3.55|4.88|0.28|
|`zq`|`*`|zng|zng-uncompressed|3.54|4.28|0.30|
|`zq`|`*`|zng|tzng|10.03|13.91|0.43|
|`zq`|`*`|zng|ndjson|55.92|68.61|1.10|
|`zq`|`*`|zng-uncompressed|zeek|14.18|21.83|0.63|
|`zq`|`*`|zng-uncompressed|zng|3.58|5.01|0.25|
|`zq`|`*`|zng-uncompressed|zng-uncompressed|3.74|4.45|0.42|
|`zq`|`*`|zng-uncompressed|tzng|9.84|13.80|0.42|
|`zq`|`*`|zng-uncompressed|ndjson|55.66|68.87|1.00|
|`zq`|`*`|tzng|zeek|16.06|40.14|1.41|
|`zq`|`*`|tzng|zng|8.27|11.96|0.63|
|`zq`|`*`|tzng|zng-uncompressed|8.21|10.95|0.57|
|`zq`|`*`|tzng|tzng|10.10|23.63|0.61|
|`zq`|`*`|tzng|ndjson|60.65|101.06|3.17|
|`zq`|`*`|ndjson|zeek|109.07|202.75|7.08|
|`zq`|`*`|ndjson|zng|104.69|170.35|5.60|
|`zq`|`*`|ndjson|zng-uncompressed|104.14|170.05|5.54|
|`zq`|`*`|ndjson|tzng|107.10|188.59|6.32|
|`zq`|`*`|ndjson|ndjson|116.35|267.95|7.80|
|`zeek-cut`||zeek|zeek-cut|1.42|1.27|0.11|
|`jq`|`-c "."`|ndjson|ndjson|40.88|5.04|1.04|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|9.39|18.25|0.89|
|`zq`|`cut ts`|zeek|zng|9.16|15.80|0.85|
|`zq`|`cut ts`|zeek|zng-uncompressed|9.20|15.96|0.79|
|`zq`|`cut ts`|zeek|tzng|9.37|17.81|0.84|
|`zq`|`cut ts`|zeek|ndjson|10.21|24.24|0.99|
|`zq`|`cut ts`|zng|zeek|4.12|6.51|0.24|
|`zq`|`cut ts`|zng|zng|4.15|5.16|0.32|
|`zq`|`cut ts`|zng|zng-uncompressed|4.01|4.77|0.38|
|`zq`|`cut ts`|zng|tzng|4.05|6.05|0.28|
|`zq`|`cut ts`|zng|ndjson|6.00|10.32|0.39|
|`zq`|`cut ts`|zng-uncompressed|zeek|3.97|6.32|0.20|
|`zq`|`cut ts`|zng-uncompressed|zng|4.05|5.01|0.37|
|`zq`|`cut ts`|zng-uncompressed|zng-uncompressed|3.97|4.72|0.37|
|`zq`|`cut ts`|zng-uncompressed|tzng|3.90|5.84|0.26|
|`zq`|`cut ts`|zng-uncompressed|ndjson|5.80|10.05|0.35|
|`zq`|`cut ts`|tzng|zeek|8.93|14.05|0.72|
|`zq`|`cut ts`|tzng|zng|8.66|11.54|0.68|
|`zq`|`cut ts`|tzng|zng-uncompressed|8.61|11.55|0.52|
|`zq`|`cut ts`|tzng|tzng|8.88|13.36|0.76|
|`zq`|`cut ts`|tzng|ndjson|9.53|19.53|0.85|
|`zq`|`cut ts`|ndjson|zeek|106.32|174.74|5.96|
|`zq`|`cut ts`|ndjson|zng|106.44|173.18|5.54|
|`zq`|`cut ts`|ndjson|zng-uncompressed|106.48|172.50|6.23|
|`zq`|`cut ts`|ndjson|tzng|104.89|172.01|5.97|
|`zq`|`cut ts`|ndjson|ndjson|107.51|182.43|6.40|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.40|1.29|0.11|
|`jq`|`-c ". \| { ts }"`|ndjson|ndjson|22.23|3.74|0.59|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|8.70|14.90|0.89|
|`zq`|`count()`|zeek|zng|8.77|15.13|0.80|
|`zq`|`count()`|zeek|zng-uncompressed|8.76|15.06|0.84|
|`zq`|`count()`|zeek|tzng|8.66|14.92|0.83|
|`zq`|`count()`|zeek|ndjson|8.68|14.87|0.86|
|`zq`|`count()`|zng|zeek|3.53|4.09|0.28|
|`zq`|`count()`|zng|zng|3.62|4.22|0.28|
|`zq`|`count()`|zng|zng-uncompressed|3.59|4.11|0.35|
|`zq`|`count()`|zng|tzng|3.52|4.03|0.37|
|`zq`|`count()`|zng|ndjson|3.48|3.99|0.32|
|`zq`|`count()`|zng-uncompressed|zeek|3.48|4.00|0.33|
|`zq`|`count()`|zng-uncompressed|zng|3.44|3.94|0.34|
|`zq`|`count()`|zng-uncompressed|zng-uncompressed|3.41|4.10|0.30|
|`zq`|`count()`|zng-uncompressed|tzng|3.42|4.10|0.32|
|`zq`|`count()`|zng-uncompressed|ndjson|3.51|4.23|0.31|
|`zq`|`count()`|tzng|zeek|8.10|10.51|0.59|
|`zq`|`count()`|tzng|zng|8.08|10.55|0.52|
|`zq`|`count()`|tzng|zng-uncompressed|8.10|10.60|0.50|
|`zq`|`count()`|tzng|tzng|8.13|10.54|0.59|
|`zq`|`count()`|tzng|ndjson|8.16|10.69|0.54|
|`zq`|`count()`|ndjson|zeek|103.56|168.64|5.55|
|`zq`|`count()`|ndjson|zng|101.89|165.64|5.82|
|`zq`|`count()`|ndjson|zng-uncompressed|104.10|169.07|5.85|
|`zq`|`count()`|ndjson|tzng|104.95|170.71|5.90|
|`zq`|`count()`|ndjson|ndjson|104.90|170.66|5.74|
|`jq`|`-c -s ". \| length"`|ndjson|ndjson|21.88|3.76|0.77|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|9.04|15.04|0.50|
|`zq`|`count() by id.orig_h`|zeek|zng|9.05|15.05|0.55|
|`zq`|`count() by id.orig_h`|zeek|zng-uncompressed|9.08|14.98|0.62|
|`zq`|`count() by id.orig_h`|zeek|tzng|9.23|15.16|0.58|
|`zq`|`count() by id.orig_h`|zeek|ndjson|9.00|14.88|0.62|
|`zq`|`count() by id.orig_h`|zng|zeek|3.89|4.11|0.06|
|`zq`|`count() by id.orig_h`|zng|zng|3.78|3.98|0.07|
|`zq`|`count() by id.orig_h`|zng|zng-uncompressed|3.75|4.00|0.03|
|`zq`|`count() by id.orig_h`|zng|tzng|3.77|3.99|0.05|
|`zq`|`count() by id.orig_h`|zng|ndjson|3.77|4.00|0.05|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zeek|3.74|3.96|0.05|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng|3.72|3.93|0.06|
|`zq`|`count() by id.orig_h`|zng-uncompressed|zng-uncompressed|3.70|3.94|0.03|
|`zq`|`count() by id.orig_h`|zng-uncompressed|tzng|3.72|3.93|0.07|
|`zq`|`count() by id.orig_h`|zng-uncompressed|ndjson|3.73|3.97|0.03|
|`zq`|`count() by id.orig_h`|tzng|zeek|8.50|10.62|0.26|
|`zq`|`count() by id.orig_h`|tzng|zng|8.49|10.65|0.22|
|`zq`|`count() by id.orig_h`|tzng|zng-uncompressed|8.39|10.48|0.26|
|`zq`|`count() by id.orig_h`|tzng|tzng|8.35|10.41|0.30|
|`zq`|`count() by id.orig_h`|tzng|ndjson|8.42|10.47|0.33|
|`zq`|`count() by id.orig_h`|ndjson|zeek|105.88|172.03|5.41|
|`zq`|`count() by id.orig_h`|ndjson|zng|107.53|174.63|5.40|
|`zq`|`count() by id.orig_h`|ndjson|zng-uncompressed|108.22|175.16|5.71|
|`zq`|`count() by id.orig_h`|ndjson|tzng|106.88|173.59|5.44|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|107.55|174.21|5.75|
|`jq`|`-c -s "group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}"`|ndjson|ndjson|20.74|3.82|0.58|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zeek|8.57|14.16|0.55|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng|8.55|14.12|0.53|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng-uncompressed|8.56|14.18|0.44|
|`zq`|`id.resp_h=52.85.83.116`|zeek|tzng|8.57|14.23|0.45|
|`zq`|`id.resp_h=52.85.83.116`|zeek|ndjson|8.53|14.02|0.59|
|`zq`|`id.resp_h=52.85.83.116`|zng|zeek|3.36|3.42|0.03|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng|3.37|3.42|0.04|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng-uncompressed|3.38|3.44|0.04|
|`zq`|`id.resp_h=52.85.83.116`|zng|tzng|3.36|3.40|0.05|
|`zq`|`id.resp_h=52.85.83.116`|zng|ndjson|3.37|3.43|0.03|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zeek|3.31|3.36|0.05|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zng|3.31|3.36|0.04|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|zng-uncompressed|3.31|3.38|0.02|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|tzng|3.30|3.37|0.02|
|`zq`|`id.resp_h=52.85.83.116`|zng-uncompressed|ndjson|3.31|3.36|0.04|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zeek|8.09|9.93|0.30|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng|8.04|9.90|0.28|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng-uncompressed|8.06|9.92|0.29|
|`zq`|`id.resp_h=52.85.83.116`|tzng|tzng|8.11|10.00|0.25|
|`zq`|`id.resp_h=52.85.83.116`|tzng|ndjson|8.15|9.97|0.34|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zeek|104.71|169.30|5.98|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng|105.02|169.71|5.73|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng-uncompressed|104.13|168.64|5.66|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|tzng|104.60|169.44|5.58|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|ndjson|102.75|166.58|5.26|
|`jq`|`-c ". \| select(.["id.resp_h"]=="52.85.83.116")"`|ndjson|ndjson|19.57|3.64|0.63|

