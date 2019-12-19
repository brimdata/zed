#!/bin/bash
set -eo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR" || exit 1

DATA="zq-sample-data"
DATA_REPO="https://github.com/mccanne/zq-sample-data.git"
if [ -d "$DATA" ]; then
  (cd "$DATA"&& git pull) || exit 1
else
  git clone "$DATA_REPO"
fi
find "$DATA" -name \*.gz -exec gunzip -f {} \;
ln -sfh zeek-default "$DATA/zeek"
ln -sfh zeek-ndjson "$DATA/ndjson"

ZEEK_LOGS="$DATA/zeek-default/*.log"
ZNG_LOGS="$DATA/zng/*.zng"
BZNG_LOGS="$DATA/bzng/*.bzng"
NDJSON_LOGS="$DATA/zeek-ndjson/*.ndjson"

TIME=$(which time)

for CMD in time zq jq zeek-cut; do
  if ! [[ $(type -P "$CMD") ]]; then
    echo "$CMD not found in PATH. Exiting."
    exit 1
  fi
done


declare -a markdowns=(
    '../performance/01_all_unmodified.md'
    '../performance/02_cut_ts.md'
    '../performance/03_count_all.md'
    '../performance/04_count_by_id_orig_h.md'
    '../performance/05_only_id_resp_h.md'
)

declare -a descriptions=(
    'Output all events unmodified'
    'Extract the field `ts`'
    'Count all events'
    'Count all events, grouped by the field `id.orig_h`'
    'Output all events containing that have field `id.resp_h` set to `52.85.83.116`'
)

declare -a zqls=(
    '*'
    'cut ts'
    'count()'
    'count() by id.orig_h'
    'id.resp_h=52.85.83.116'
)

declare -a jqs=(
    '.'
    '. | { ts }'
    '. | length'
    'group_by(."id.orig_h")[] | length as $l | .[0] | .count = $l | {count,"id.orig_h"}'
    '. | select(.["id.resp_h"]=="52.85.83.116")'
)
declare -a jqflags=(
    '-c'
    '-c'
    '-c -s'
    '-c -s'
    '-c'
)
declare -a zcuts=(
    ''
    'ts'
    'NONE'
    'NONE'
    'NONE'
)

for (( n=0; n<"${#zqls[@]}"; n++ ))
do
    desc=${descriptions[$n]}
    MD=${markdowns[$n]}
    zql=${zqls[$n]}
    echo -e "#### $desc\n" | tee "$MD"
    echo "|**<br>Tool**|**<br>Arguments**|**Input<br>Format**|**Output<br>Format**|**<br>Real**|**<br>User**|**<br>Sys**|" | tee -a "$MD"
    echo "|:----------:|:---------------:|:-----------------:|:------------------:|-----------:|-----------:|----------:|" | tee -a "$MD"
    for INPUT in bzng ; do   # zeek bzng zng ndjson
      for OUTPUT in bzng; do
        echo -n "|\`zq\`|\`$zql\`|$INPUT|$OUTPUT|" | tee -a "$MD"
        ALL_TIMES=$(($TIME zq -i "$INPUT" -f "$OUTPUT" "$zql" $DATA/$INPUT/* > /dev/null) 2>&1)
        echo $ALL_TIMES | awk '{ print $1 "|" $3 "|" $5 "|" }' | tee -a "$MD"
      done
    done

    zcut=${zcuts[$n]}
    if [[ $zcut != "NONE" ]]; then
      echo "|\`zeek-cut\`|\`$zcut\`|zeek|zeek-cut|" | sed 's/\`\`//' | tr -d '\n' | tee -a "$MD"
      ALL_TIMES=$(($TIME cat $ZEEK_LOGS | zeek-cut $zcut > /dev/null) 2>&1)
      echo $ALL_TIMES | awk '{ print $1 "|" $3 "|" $5 "|" }' | tee -a "$MD"
    fi

    jq=${jqs[$n]}
    jqflag=${jqflags[$n]}
    echo -n "|\`jq\`|\`$jqflag \"${jq//|/\\|}\"\`|ndjson|ndjson|" | tee -a "$MD"
    ALL_TIMES=$(($TIME jq $jqflag "$jq" $NDJSON_LOGS > /dev/null) 2>&1)
    echo $ALL_TIMES | awk '{ print $1 "|" $3 "|" $5 "|" }' | tee -a "$MD"

    echo
done


