# Performance

The tables below provide a summary of simple operations and how `zq`
performs at them relative to `zeek-cut` and `jq`. All operations were performed
on a Google Cloud `n1-standard-8` VM (8 vCPUs, 30 GB memory) with the logs
stored on a local SSD. `make perf-compare` was used to generate the results.

As there are many results to sift through, here's a few key summary take-aways:

* If all you care about is cutting field values by column, `zeek-cut` does still perform the best. (Alas, that's all `zeek-cut` can do. :smiley:)

* The numerous input/output formats in `zq` are helpful for fitting into your legacy pipelines. However, ZNG performs the best of all `zq`-compatible formats, due to its binary/optimized nature. If you have logs in a non-ZNG format and expect to query them many times, a one-time pass through `zq` to convert them to ZNG format will save you significant time.

* Particularly when working in ZNG format & when simple analytics (counting, grouping) are in play, `zq` can significantly outperform `jq`. That said, `zq` does not (yet) include the full set of mathematical/other operations available in `jq`. If there's glaring functional omisssions that are limiting your use of `zq`, we welcome [contributions](../README.md#contributing).

# Results

### Output all events unmodified

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`*`|zeek|zeek|15.53|31.54|1.16|
|`zq`|`*`|zeek|zng|6.16|8.18|0.44|
|`zq`|`*`|zeek|tzng|10.36|20.23|0.63|
|`zq`|`*`|zeek|ndjson|67.83|93.00|4.13|
|`zq`|`*`|zng|zeek|14.56|20.83|0.55|
|`zq`|`*`|zng|zng|3.23|3.80|0.35|
|`zq`|`*`|zng|tzng|10.33|13.96|0.44|
|`zq`|`*`|zng|ndjson|63.03|71.85|1.05|
|`zq`|`*`|tzng|zeek|15.43|33.72|1.09|
|`zq`|`*`|tzng|zng|8.09|10.06|0.53|
|`zq`|`*`|tzng|tzng|10.20|21.66|0.46|
|`zq`|`*`|tzng|ndjson|67.35|94.67|3.91|
|`zq`|`*`|ndjson|zeek|56.26|96.27|4.34|
|`zq`|`*`|ndjson|zng|52.16|70.37|3.41|
|`zq`|`*`|ndjson|tzng|54.33|87.20|3.70|
|`zq`|`*`|ndjson|ndjson|72.23|161.66|5.48|
|`zeek-cut`||zeek|zeek-cut|1.40|1.30|0.07|
|`jq`|`-c "."`|ndjson|ndjson|40.27|5.01|0.84|

### Extract the field `ts`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`cut ts`|zeek|zeek|6.70|10.46|0.55|
|`zq`|`cut ts`|zeek|zng|6.45|8.29|0.46|
|`zq`|`cut ts`|zeek|tzng|6.63|10.02|0.49|
|`zq`|`cut ts`|zeek|ndjson|7.68|16.89|0.55|
|`zq`|`cut ts`|zng|zeek|3.76|5.94|0.20|
|`zq`|`cut ts`|zng|zng|3.72|4.50|0.32|
|`zq`|`cut ts`|zng|tzng|3.73|5.71|0.19|
|`zq`|`cut ts`|zng|ndjson|6.61|10.85|0.34|
|`zq`|`cut ts`|tzng|zeek|8.80|12.80|0.57|
|`zq`|`cut ts`|tzng|zng|8.47|10.51|0.50|
|`zq`|`cut ts`|tzng|tzng|8.74|12.35|0.43|
|`zq`|`cut ts`|tzng|ndjson|9.64|19.15|0.58|
|`zq`|`cut ts`|ndjson|zeek|52.62|72.58|3.43|
|`zq`|`cut ts`|ndjson|zng|52.15|70.09|3.46|
|`zq`|`cut ts`|ndjson|tzng|52.46|71.97|3.51|
|`zq`|`cut ts`|ndjson|ndjson|53.59|80.07|3.82|
|`zeek-cut`|`ts`|zeek|zeek-cut|1.33|1.23|0.09|
|`jq`|`-c ". \| { ts }"`|ndjson|ndjson|22.52|3.80|0.61|

### Count all events

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count()`|zeek|zeek|5.55|5.98|0.18|
|`zq`|`count()`|zeek|zng|5.60|6.03|0.20|
|`zq`|`count()`|zeek|tzng|5.65|6.09|0.22|
|`zq`|`count()`|zeek|ndjson|5.54|5.93|0.23|
|`zq`|`count()`|zng|zeek|2.99|3.03|0.02|
|`zq`|`count()`|zng|zng|2.97|2.98|0.04|
|`zq`|`count()`|zng|tzng|2.97|2.99|0.04|
|`zq`|`count()`|zng|ndjson|2.98|2.96|0.07|
|`zq`|`count()`|tzng|zeek|7.89|8.52|0.21|
|`zq`|`count()`|tzng|zng|7.86|8.30|0.30|
|`zq`|`count()`|tzng|tzng|7.84|8.28|0.29|
|`zq`|`count()`|tzng|ndjson|7.81|8.31|0.23|
|`zq`|`count()`|ndjson|zeek|50.80|59.57|2.37|
|`zq`|`count()`|ndjson|zng|50.39|59.20|2.54|
|`zq`|`count()`|ndjson|tzng|50.14|58.85|2.26|
|`zq`|`count()`|ndjson|ndjson|50.31|58.81|2.50|
|`jq`|`-c -s ". \| length"`|ndjson|ndjson|21.88|3.84|0.55|

### Count all events, grouped by the field `id.orig_h`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`count() by id.orig_h`|zeek|zeek|5.93|6.44|0.20|
|`zq`|`count() by id.orig_h`|zeek|zng|5.96|6.57|0.15|
|`zq`|`count() by id.orig_h`|zeek|tzng|6.02|6.57|0.22|
|`zq`|`count() by id.orig_h`|zeek|ndjson|5.90|6.40|0.22|
|`zq`|`count() by id.orig_h`|zng|zeek|3.33|3.36|0.03|
|`zq`|`count() by id.orig_h`|zng|zng|3.31|3.34|0.04|
|`zq`|`count() by id.orig_h`|zng|tzng|3.35|3.37|0.04|
|`zq`|`count() by id.orig_h`|zng|ndjson|3.32|3.37|0.02|
|`zq`|`count() by id.orig_h`|tzng|zeek|8.08|8.54|0.35|
|`zq`|`count() by id.orig_h`|tzng|zng|8.11|8.76|0.22|
|`zq`|`count() by id.orig_h`|tzng|tzng|8.06|8.63|0.23|
|`zq`|`count() by id.orig_h`|tzng|ndjson|8.01|8.63|0.18|
|`zq`|`count() by id.orig_h`|ndjson|zeek|50.75|61.33|2.66|
|`zq`|`count() by id.orig_h`|ndjson|zng|50.71|61.54|2.49|
|`zq`|`count() by id.orig_h`|ndjson|tzng|50.51|61.20|2.68|
|`zq`|`count() by id.orig_h`|ndjson|ndjson|50.85|61.67|2.54|
|`jq`|`-c -s "group_by(."id.orig_h")[] \| length as $l \| .[0] \| .count = $l \| {count,"id.orig_h"}"`|ndjson|ndjson|21.06|3.75|0.51|

### Output all events with the field `id.resp_h` set to `52.85.83.116`

|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|
|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zeek|5.86|6.32|0.19|
|`zq`|`id.resp_h=52.85.83.116`|zeek|zng|5.83|6.20|0.22|
|`zq`|`id.resp_h=52.85.83.116`|zeek|tzng|5.73|6.13|0.19|
|`zq`|`id.resp_h=52.85.83.116`|zeek|ndjson|5.68|6.14|0.12|
|`zq`|`id.resp_h=52.85.83.116`|zng|zeek|2.99|2.99|0.02|
|`zq`|`id.resp_h=52.85.83.116`|zng|zng|2.99|2.97|0.03|
|`zq`|`id.resp_h=52.85.83.116`|zng|tzng|2.96|2.95|0.03|
|`zq`|`id.resp_h=52.85.83.116`|zng|ndjson|2.97|2.94|0.04|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zeek|7.70|8.10|0.28|
|`zq`|`id.resp_h=52.85.83.116`|tzng|zng|7.75|8.28|0.21|
|`zq`|`id.resp_h=52.85.83.116`|tzng|tzng|7.84|8.35|0.23|
|`zq`|`id.resp_h=52.85.83.116`|tzng|ndjson|7.77|8.23|0.23|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zeek|50.08|58.62|2.33|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|zng|50.29|58.95|2.53|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|tzng|50.43|58.98|2.47|
|`zq`|`id.resp_h=52.85.83.116`|ndjson|ndjson|50.28|58.87|2.55|
|`jq`|`-c ". \| select(.["id.resp_h"]=="52.85.83.116")"`|ndjson|ndjson|19.56|3.66|0.52|

