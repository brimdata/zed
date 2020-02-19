# Performance

The tables below provide a summary of simple operations and how `zq`
performs at them relative to `zeek-cut` and `jq`. All operations were performed
on a Google Cloud `n1-standard-8` VM (8 vCPUs, 30 GB memory) with the logs
stored on a local SSD. `make perf-compare` was used to generate the results.

As there are many results to sift through, here's a few key summary take-aways:

* If all you care about is cutting field values by column, `zeek-cut` does still perform the best. (Alas, that's all `zeek-cut` can do. :smiley:)

* The numerous input/output formats in `zq` are helpful for fitting into your legacy pipelines. However, BZNG performs the best of all `zq`-compatible formats, due to its binary/optimized nature. If you have logs in a non-BZNG format and expect to query them many times, a one-time pass through `zq` to convert them to BZNG format will save you significant time.

* Particularly when working in BZNG format & when simple analytics (counting, grouping) are in play, `zq` can significantly outperform `jq`. That said, `zq` does not (yet) include the full set of mathematical/other operations available in `jq`. If there's glaring functional omisssions that are limiting your use of `zq`, we welcome [contributions](../README.md#contributing).

# Results

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|18.25|35.86|1.72|
|`zq`|`*`|zeek|bzng|5.08|6.85|0.66|
|`zq`|`*`|zeek|zng|12.73|23.44|1.02|
|`zq`|`*`|zeek|ndjson|64.13|88.97|4.47|
|`zq`|`*`|bzng|zeek|16.05|21.72|0.70|
|`zq`|`*`|bzng|bzng|1.67|2.30|0.28|
|`zq`|`*`|bzng|zng|11.45|15.10|0.56|
|`zq`|`*`|bzng|ndjson|58.98|64.94|1.08|
|`zq`|`*`|zng|zeek|18.43|37.60|1.77|
|`zq`|`*`|zng|bzng|6.24|8.16|0.73|
|`zq`|`*`|zng|zng|12.86|25.14|0.99|
|`zq`|`*`|zng|ndjson|63.73|89.58|4.50|
|`zq`|`*`|ndjson|zeek|31.94|75.65|3.86|
|`zq`|`*`|ndjson|bzng|27.21|43.65|2.85|
|`zq`|`*`|ndjson|zng|29.88|64.28|3.34|
|`zq`|`*`|ndjson|ndjson|61.20|119.14|6.17|
|`zeek-cut`|``|zeek|zeek-cut|0.27|0.00|0.22|
|`jq`|`-c "."`|ndjson|ndjson|38.63|38.18|0.31|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|5.73|9.06|0.60|
|`zq`|`cut ts`|zeek|bzng|5.60|7.61|0.57|
|`zq`|`cut ts`|zeek|zng|5.82|9.81|0.58|
|`zq`|`cut ts`|zeek|ndjson|6.56|15.46|0.54|
|`zq`|`cut ts`|bzng|zeek|2.04|3.65|0.20|
|`zq`|`cut ts`|bzng|bzng|2.24|2.91|0.38|
|`zq`|`cut ts`|bzng|zng|2.12|4.22|0.11|
|`zq`|`cut ts`|bzng|ndjson|4.94|8.26|0.45|
|`zq`|`cut ts`|zng|zeek|6.91|10.43|0.61|
|`zq`|`cut ts`|zng|bzng|6.80|8.90|0.67|
|`zq`|`cut ts`|zng|zng|6.99|11.09|0.68|
|`zq`|`cut ts`|zng|ndjson|7.88|17.23|0.65|
|`zq`|`cut ts`|ndjson|zeek|28.16|45.73|3.09|
|`zq`|`cut ts`|ndjson|bzng|28.15|44.85|2.64|
|`zq`|`cut ts`|ndjson|zng|28.20|46.24|2.94|
|`zq`|`cut ts`|ndjson|ndjson|29.15|53.84|3.44|
|`zeek-cut`|`ts`|zeek|zeek-cut|0.30|0.00|0.24|
|`jq`|`-c ". \| { ts }"`|ndjson|ndjson|20.43|20.27|0.15|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|4.56|5.00|0.20|
|`zq`|`count()`|zeek|bzng|4.49|4.87|0.21|
|`zq`|`count()`|zeek|zng|4.60|5.00|0.19|
|`zq`|`count()`|zeek|ndjson|4.57|4.93|0.26|
|`zq`|`count()`|bzng|zeek|1.31|1.29|0.06|
|`zq`|`count()`|bzng|bzng|1.32|1.31|0.05|
|`zq`|`count()`|bzng|zng|1.33|1.32|0.04|
|`zq`|`count()`|bzng|ndjson|1.33|1.31|0.06|
|`zq`|`count()`|zng|zeek|5.88|6.29|0.32|
|`zq`|`count()`|zng|bzng|5.74|6.25|0.22|
|`zq`|`count()`|zng|zng|5.74|6.16|0.24|
|`zq`|`count()`|zng|ndjson|5.80|6.18|0.28|
|`zq`|`count()`|ndjson|zeek|25.72|31.20|2.10|
|`zq`|`count()`|ndjson|bzng|25.61|31.31|1.83|
|`zq`|`count()`|ndjson|zng|25.52|31.18|1.99|
|`zq`|`count()`|ndjson|ndjson|25.42|30.99|1.89|
|`jq`|`-c -s ". \| length"`|ndjson|ndjson|23.54|20.95|2.58|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|5.00|5.55|0.24|
|`zq`|`count() by id.orig_h`|zeek|bzng|5.02|5.53|0.21|
|`zq`|`count() by id.orig_h`|zeek|zng|4.98|5.49|0.22|
|`zq`|`count() by id.orig_h`|zeek|ndjson|5.02|5.55|0.25|
|`zq`|`count() by id.orig_h`|bzng|zeek|1.77|1.77|0.05|
|`zq`|`count() by id.orig_h`|bzng|bzng|1.74|1.77|0.03|
|`zq`|`count() by id.orig_h`|bzng|zng|1.77|1.75|0.06|
|`zq`|`count() by id.orig_h`|bzng|ndjson|1.77|1.74|0.08|
|`zq`|`count() by id.orig_h`|zng|zeek|6.27|6.91|0.21|
|`zq`|`count() by id.orig_h`|zng|bzng|6.33|6.95|0.27|
|`zq`|`count() by id.orig_h`|zng|zng|6.20|6.74|0.27|
|`zq`|`count() by id.orig_h`|zng|ndjson|6.24|6.81|0.24|
|`zq`|`count() by id.orig_h`|ndjson|zeek|26.51|34.23|2.29|
|`zq`|`count() by id.orig_h`|ndjson|bzng|26.39|34.09|2.29|
|`zq`|`count() by id.orig_h`|ndjson|zng|26.49|34.13|2.23|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|26.44|33.84|2.29|
|`jq`|`-c -s "group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}"`|ndjson|ndjson|34.10|31.77|2.32|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zeek|4.75|5.16|0.22|
|`zq`|`id.resp_h=52.85.83.116`|zeek|bzng|4.76|5.08|0.26|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng|4.73|5.15|0.22|
|`zq`|`id.resp_h=52.85.83.116`|zeek|ndjson|4.76|5.14|0.21|
|`zq`|`id.resp_h=52.85.83.116`|bzng|zeek|1.42|1.39|0.04|
|`zq`|`id.resp_h=52.85.83.116`|bzng|bzng|1.40|1.35|0.06|
|`zq`|`id.resp_h=52.85.83.116`|bzng|zng|1.40|1.36|0.05|
|`zq`|`id.resp_h=52.85.83.116`|bzng|ndjson|1.39|1.37|0.03|
|`zq`|`id.resp_h=52.85.83.116`|zng|zeek|5.89|6.32|0.21|
|`zq`|`id.resp_h=52.85.83.116`|zng|bzng|6.05|6.43|0.27|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng|5.95|6.30|0.34|
|`zq`|`id.resp_h=52.85.83.116`|zng|ndjson|5.98|6.47|0.22|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zeek|25.78|31.27|1.88|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|bzng|25.70|31.09|2.09|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng|25.46|30.89|1.93|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|ndjson|25.76|31.23|2.19|
|`jq`|`-c ". \| select(.["id.resp_h"]=="52.85.83.116")"`|ndjson|ndjson|17.94|17.75|0.18|

