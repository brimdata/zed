zapi new -k archivestore -d s3://brim-scratch/mark/sp-m1 -thresh 5MB sp-m1
#
# This is the same smtp.log from zq-sample-data
#
zapi -s sp-m1 post s3://brim-scratch/mark/conn.log.gz
#
# The count() from zq should be identical to 
# the count() from zapi get -chunk
#
zapi -s sp-m1 get -p 2 -t "count()"
zapi -s sp-m1 get -p 2 -t "39161"