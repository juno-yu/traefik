TODO

```
docker run -v "$PWD":/dynamodb_local_db -p 8000:8000 cnadiminti/dynamodb-local:2020-10-12
docker run -it -v $(pwd)/dist:/opt/mybin alpine /bin/sh

/opt/mybin # ./traefik  --help | grep dynamo
    --providers.dynamodb  (Default: "false")
    --providers.dynamodb.accesskeyid  (Default: "")
    --providers.dynamodb.endpoint  (Default: "")
        The endpoint of a dynamodb. Used for testing with a local dynamodb
    --providers.dynamodb.refreshseconds  (Default: "15")
    --providers.dynamodb.region  (Default: "")
    --providers.dynamodb.secretaccesskey  (Default: "")
    --providers.dynamodb.tablename  (Default: "")
        The AWS dynamodb table that stores configuration for traefik
/opt/mybin # ./traefik --providers.dynamodb --providers.dynamodb.endpoint=http://0.0.0.0:8000 --providers.dynamodb.tablename=myTable
```