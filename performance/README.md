# Performance

The tables below provide a summary of simple operations and how `zq`
performs at them relative to `zeek-cut` and `jq`. All operations were performed
on a Google Cloud `n1-standard-8` VM (8 vCPUs, 30 GB memory) with the logs
stored on a local SSD. The [`comparison-test.sh`](../scripts/comparison-test.sh)
script (which uses the [zq-sample-data](https://github.com/mccanne/zq-sample-data))
was used to generate the results.

As there are many results to sift through, here's a few key summary take-aways:

* If all you care about is cutting field values by column, `zeek-cut` does still perform the best. (Alas, that's all `zeek-cut` can do. :smiley:)

* The numerous input/output formats in `zq` are helpful for fitting into your legacy pipelines. However, BZNG performs the best of all `zq`-compatible formats, due to its binary/optimized nature. If you have logs in a non-BZNG format and expect to query them many times, a one-time pass through `zq` to convert them to BZNG format will save you significant time.

* Particularly when working in BZNG format & when simple analytics (counting, gropuing) are in play, `zq` can significantly outperform `jq`. That said, `zq` does not (yet) include the full set of mathematical/other operations available in `jq`. If there's glaring functional omisssions that are limiting your use of `zq`, we welcome [contributions](../README.md#contributing).

# Results

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|18.14|35.58|1.99|
|`zq`|`*`|zeek|bzng|5.13|7.09|0.52|
|`zq`|`*`|zeek|zng|13.12|23.91|1.05|
|`zq`|`*`|zeek|ndjson|64.03|88.43|4.61|
|`zq`|`*`|bzng|zeek|15.66|21.19|0.63|
|`zq`|`*`|bzng|bzng|1.64|2.18|0.32|
|`zq`|`*`|bzng|zng|11.67|15.20|0.50|
|`zq`|`*`|bzng|ndjson|58.78|64.60|0.92|
|`zq`|`*`|ndjson|zeek|31.99|75.74|3.78|
|`zq`|`*`|ndjson|bzng|27.20|43.48|2.84|
|`zq`|`*`|ndjson|zng|30.08|65.00|3.22|
|`zq`|`*`|ndjson|ndjson|61.15|118.48|5.94|
|`zeek-cut`|``|zeek|zeek-cut|0.26|0.00|0.22|
|`jq`|`-c "."`|ndjson|ndjson|38.41|38.01|0.33|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|5.79|8.86|0.52|
|`zq`|`cut ts`|zeek|bzng|5.77|8.10|0.62|
|`zq`|`cut ts`|zeek|zng|6.10|10.75|0.70|
|`zq`|`cut ts`|zeek|ndjson|6.37|14.98|0.56|
|`zq`|`cut ts`|bzng|zeek|2.17|3.57|0.29|
|`zq`|`cut ts`|bzng|bzng|2.19|2.86|0.35|
|`zq`|`cut ts`|bzng|zng|2.08|4.26|0.12|
|`zq`|`cut ts`|bzng|ndjson|5.15|8.55|0.43|
|`zq`|`cut ts`|ndjson|zeek|28.19|45.57|2.87|
|`zq`|`cut ts`|ndjson|bzng|27.85|43.84|2.78|
|`zq`|`cut ts`|ndjson|zng|27.77|45.55|2.81|
|`zq`|`cut ts`|ndjson|ndjson|28.65|52.70|3.10|
|`zeek-cut`|`ts`|zeek|zeek-cut|0.29|0.00|0.23|
|`jq`|`-c ". \| { ts }"`|ndjson|ndjson|21.02|20.83|0.19|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|4.62|4.99|0.24|
|`zq`|`count()`|zeek|bzng|4.65|5.05|0.18|
|`zq`|`count()`|zeek|zng|4.66|5.10|0.17|
|`zq`|`count()`|zeek|ndjson|4.58|4.95|0.23|
|`zq`|`count()`|bzng|zeek|1.32|1.32|0.05|
|`zq`|`count()`|bzng|bzng|1.32|1.30|0.05|
|`zq`|`count()`|bzng|zng|1.32|1.32|0.04|
|`zq`|`count()`|bzng|ndjson|1.31|1.31|0.04|
|`zq`|`count()`|ndjson|zeek|25.99|31.68|1.85|
|`zq`|`count()`|ndjson|bzng|25.39|30.85|1.89|
|`zq`|`count()`|ndjson|zng|25.64|31.08|2.01|
|`zq`|`count()`|ndjson|ndjson|25.40|31.01|1.80|
|`jq`|`-c -s ". \| length"`|ndjson|ndjson|24.24|21.47|2.77|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|5.19|5.78|0.23|
|`zq`|`count() by id.orig_h`|zeek|bzng|5.15|5.65|0.21|
|`zq`|`count() by id.orig_h`|zeek|zng|5.07|5.53|0.22|
|`zq`|`count() by id.orig_h`|zeek|ndjson|5.10|5.56|0.25|
|`zq`|`count() by id.orig_h`|bzng|zeek|1.78|1.80|0.04|
|`zq`|`count() by id.orig_h`|bzng|bzng|1.78|1.75|0.08|
|`zq`|`count() by id.orig_h`|bzng|zng|1.80|1.82|0.03|
|`zq`|`count() by id.orig_h`|bzng|ndjson|1.80|1.82|0.04|
|`zq`|`count() by id.orig_h`|ndjson|zeek|26.36|33.83|2.11|
|`zq`|`count() by id.orig_h`|ndjson|bzng|26.70|34.26|2.22|
|`zq`|`count() by id.orig_h`|ndjson|zng|26.67|34.18|2.02|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|26.34|33.75|2.14|
|`jq`|`-c -s "group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}"`|ndjson|ndjson|35.80|32.87|2.93|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zeek|4.99|5.27|0.28|
|`zq`|`id.resp_h=52.85.83.116`|zeek|bzng|4.98|5.40|0.20|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng|4.98|5.29|0.26|
|`zq`|`id.resp_h=52.85.83.116`|zeek|ndjson|4.98|5.32|0.27|
|`zq`|`id.resp_h=52.85.83.116`|bzng|zeek|1.44|1.39|0.05|
|`zq`|`id.resp_h=52.85.83.116`|bzng|bzng|1.44|1.40|0.05|
|`zq`|`id.resp_h=52.85.83.116`|bzng|zng|1.42|1.38|0.05|
|`zq`|`id.resp_h=52.85.83.116`|bzng|ndjson|1.44|1.39|0.06|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zeek|26.34|31.80|1.85|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|bzng|26.32|31.77|1.94|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng|26.22|31.67|2.06|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|ndjson|26.08|31.44|2.08|
|`jq`|`-c ". \| select(.["id.resp_h"]=="52.85.83.116")"`|ndjson|ndjson|18.47|18.27|0.20|

