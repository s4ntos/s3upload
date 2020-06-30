# s3upload
Golang s3 upload

Small script to upload to AWS S3 with SSE encryption. 
The script only required List and Put privileges and be allowed to use the key for encryption.

Sample configuration file
```
Normal configuration file used
Region: "eu-west-1"
Access_key : "A1234567890"
Secret_key : "AAAAAA1111111122222222333333333333"
Bucket: "bucketname"
Folder:  "directory"
ACL: "bucket-owner-full-control" # Optional
EncryptionKey: "" # Optional
Locations : [ "./test" ]
Filter : "file.*.csv$" # Regex
Upload_existing : true
Archive: true
Remove: true
RunOnce: false
```
