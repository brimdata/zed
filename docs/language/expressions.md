# Expressions

> **Note:** The example below was generated using the
> [educational sample data](https://github.com/brimdata/zed-sample-data/tree/edu-data/edu),
> which you may wish to clone locally to reproduce the example and create
> your own query variations.

Comprehensive documentation for Zed expressions is still a work in progress. In
the meantime, here's an example expression with simple math to get started:

```mdtest-command dir=zed-sample-data/edu/zson
zq -f table 'AvgScrMath != null | put combined_scores:=AvgScrMath+AvgScrRead+AvgScrWrite | cut sname,combined_scores,AvgScrMath,AvgScrRead,AvgScrWrite | head 5' testscores.zson
```

#### Output:
```mdtest-output
sname                       combined_scores AvgScrMath AvgScrRead AvgScrWrite
APEX Academy                1115            371        376        368
ARISE High                  1095            367        359        369
Abraham Lincoln High        1464            491        489        484
Abraham Lincoln Senior High 1319            462        432        425
Academia Avance Charter     1148            386        380        382
```
