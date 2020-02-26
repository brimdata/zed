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
|`zq`|`*`|zeek|zeek|15.11|30.85|1.15|
|`zq`|`*`|zeek|bzng|19.98|37.75|1.51|
|`zq`|`*`|zeek|zng|9.97|19.12|0.68|
|`zq`|`*`|zeek|ndjson|64.82|89.87|4.06|
|`zq`|`*`|bzng|zeek|13.86|19.95|0.64|
|`zq`|`*`|bzng|bzng|18.17|25.15|0.71|
|`zq`|`*`|bzng|zng|9.81|13.31|0.44|
|`zq`|`*`|bzng|ndjson|59.80|68.63|1.08|
|`zq`|`*`|zng|zeek|15.08|32.96|1.22|
|`zq`|`*`|zng|bzng|20.21|40.64|1.49|
|`zq`|`*`|zng|zng|10.07|21.56|0.41|
|`zq`|`*`|zng|ndjson|65.03|92.40|3.95|
|`zq`|`*`|ndjson|zeek|54.71|94.68|4.39|
|`zq`|`*`|ndjson|bzng|54.85|95.81|4.28|
|`zq`|`*`|ndjson|zng|53.63|86.94|3.78|
|`zq`|`*`|ndjson|ndjson|71.11|160.53|5.69|
|`zeek-cut`|(none)|zeek|zeek-cut|1.34|1.25|0.07|
|`jq`|`-c "."`|ndjson|ndjson|39.47|4.98|0.95|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|6.68|10.53|0.56|
|`zq`|`cut ts`|zeek|bzng|6.49|9.50|0.58|
|`zq`|`cut ts`|zeek|zng|6.50|9.88|0.50|
|`zq`|`cut ts`|zeek|ndjson|7.37|16.41|0.59|
|`zq`|`cut ts`|bzng|zeek|3.58|5.65|0.20|
|`zq`|`cut ts`|bzng|bzng|3.64|5.17|0.29|
|`zq`|`cut ts`|bzng|zng|3.59|5.39|0.19|
|`zq`|`cut ts`|bzng|ndjson|6.49|10.46|0.42|
|`zq`|`cut ts`|zng|zeek|8.56|12.53|0.60|
|`zq`|`cut ts`|zng|bzng|8.45|11.73|0.55|
|`zq`|`cut ts`|zng|zng|8.64|12.20|0.59|
|`zq`|`cut ts`|zng|ndjson|9.52|19.13|0.63|
|`zq`|`cut ts`|ndjson|zeek|52.35|72.86|3.54|
|`zq`|`cut ts`|ndjson|bzng|51.90|72.00|3.34|
|`zq`|`cut ts`|ndjson|zng|52.13|72.58|3.51|
|`zq`|`cut ts`|ndjson|ndjson|53.06|79.90|3.88|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.30|1.23|0.07|
|`jq`|`-c ". \| { ts }"`|ndjson|ndjson|21.55|3.73|0.56|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|5.20|5.68|0.13|
|`zq`|`count()`|zeek|bzng|5.19|5.66|0.14|
|`zq`|`count()`|zeek|zng|5.19|5.66|0.15|
|`zq`|`count()`|zeek|ndjson|5.21|5.64|0.23|
|`zq`|`count()`|bzng|zeek|2.72|2.71|0.06|
|`zq`|`count()`|bzng|bzng|2.75|2.77|0.03|
|`zq`|`count()`|bzng|zng|2.74|2.77|0.02|
|`zq`|`count()`|bzng|ndjson|2.71|2.75|0.02|
|`zq`|`count()`|zng|zeek|7.18|7.58|0.38|
|`zq`|`count()`|zng|bzng|7.14|7.53|0.32|
|`zq`|`count()`|zng|zng|7.22|7.73|0.20|
|`zq`|`count()`|zng|ndjson|7.32|7.82|0.21|
|`zq`|`count()`|ndjson|zeek|48.57|57.31|2.61|
|`zq`|`count()`|ndjson|bzng|49.73|58.64|2.48|
|`zq`|`count()`|ndjson|zng|49.06|57.87|2.48|
|`zq`|`count()`|ndjson|ndjson|49.78|58.74|2.57|
|`jq`|`-c -s ". \| length"`|ndjson|ndjson|21.34|3.76|0.60|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|5.82|6.41|0.13|
|`zq`|`count() by id.orig_h`|zeek|bzng|5.87|6.46|0.14|
|`zq`|`count() by id.orig_h`|zeek|zng|5.87|6.41|0.18|
|`zq`|`count() by id.orig_h`|zeek|ndjson|5.78|6.36|0.15|
|`zq`|`count() by id.orig_h`|bzng|zeek|3.25|3.29|0.03|
|`zq`|`count() by id.orig_h`|bzng|bzng|3.23|3.28|0.02|
|`zq`|`count() by id.orig_h`|bzng|zng|3.23|3.24|0.06|
|`zq`|`count() by id.orig_h`|bzng|ndjson|3.24|3.27|0.04|
|`zq`|`count() by id.orig_h`|zng|zeek|7.77|8.46|0.14|
|`zq`|`count() by id.orig_h`|zng|bzng|7.78|8.38|0.29|
|`zq`|`count() by id.orig_h`|zng|zng|7.76|8.43|0.22|
|`zq`|`count() by id.orig_h`|zng|ndjson|7.76|8.43|0.16|
|`zq`|`count() by id.orig_h`|ndjson|zeek|50.57|61.81|2.59|
|`zq`|`count() by id.orig_h`|ndjson|bzng|50.33|61.51|2.49|
|`zq`|`count() by id.orig_h`|ndjson|zng|50.14|61.01|2.86|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|50.42|61.58|2.64|
|`jq`|`-c -s "group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}"`|ndjson|ndjson|20.57|3.59|0.68|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zeek|5.58|6.02|0.15|
|`zq`|`id.resp_h=52.85.83.116`|zeek|bzng|5.64|6.04|0.20|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng|5.69|6.12|0.18|
|`zq`|`id.resp_h=52.85.83.116`|zeek|ndjson|5.60|6.03|0.18|
|`zq`|`id.resp_h=52.85.83.116`|bzng|zeek|2.92|2.92|0.02|
|`zq`|`id.resp_h=52.85.83.116`|bzng|bzng|2.93|2.90|0.05|
|`zq`|`id.resp_h=52.85.83.116`|bzng|zng|2.95|2.94|0.03|
|`zq`|`id.resp_h=52.85.83.116`|bzng|ndjson|2.90|2.88|0.03|
|`zq`|`id.resp_h=52.85.83.116`|zng|zeek|7.60|8.05|0.26|
|`zq`|`id.resp_h=52.85.83.116`|zng|bzng|7.58|8.16|0.22|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng|7.56|8.08|0.19|
|`zq`|`id.resp_h=52.85.83.116`|zng|ndjson|7.64|8.00|0.36|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zeek|50.00|58.80|2.40|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|bzng|50.12|58.99|2.52|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng|49.73|58.63|2.59|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|ndjson|49.34|58.02|2.68|
|`jq`|`-c ". \| select(.["id.resp_h"]=="52.85.83.116")"`|ndjson|ndjson|18.97|3.56|0.60|

