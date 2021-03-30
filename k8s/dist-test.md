# Testing procedures for distributed zqd

These are working notes and instructions for how to test zqd for distributed deployments.

## Settng up a basic test with zapi and zqd

### Set up a basic test of parallel search using multiple zqd processes

This is a test for a local machine.

#### 1. Create a working directory, copy in a zeek archive, and create a dir for the spaces, then start zqd:
```
mkdir ./testp
cd testp
cp zq/zed-sample-data/zeek-default/conn.log.gz .
mkdir spaces
zqd listen -data spaces
```
#### 2. Now use zapi to create a space and post (import) the zeek log to a zar archive:
```
zapi new -k archivestore -thresh 10MB conn-sp
zapi -s conn-sp post conn.log.gz
```
Because of the `-thresh` parameter, this will split the zeek log into several 10MB chunks.

#### 3. Test a typical query
```
zapi -s conn-sp get -t "count()"
```

#### 4. Using zapi, test a zqd "worker" style query that specifies just one chunk
A new parameter to zapi, `-chunk`, specifies which part of the zar archive that instance of zqd should read.

The -chunk parameter is formatted similarly to the file names for easy copy and paste, e.g.
```
zapi -s conn-sp get -t -chunk d-1iKffGi1BLPNF7xQypzXXtOUw3g-242597-1521912652111597000-1521912320525896000 "count()"

zapi -s conn-sp get -t -chunk d-1iKffF457VH57kNsoZdl75hix1v-246201-1521912990158539000-1521912652111698000 "count()"
```
