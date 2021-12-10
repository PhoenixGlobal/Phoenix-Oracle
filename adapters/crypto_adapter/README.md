# CryptoCompare External Adapter

Adapter for use on Google Cloud Platform, AWS Lambda or Docker. Upload Zip and use trigger URL as bridge endpoint.

## Install

### Build yourself

```bash
npm install
```

Create zip:

```bash
zip -r cl-cc.zip .
```

### Use precompiled release

Use one of our precompiled ZIP files from [Releases](https://github.com/OracleFinder/CryptoCompareExternalAdapter/releases). Most recent release: [cl-cc-aws-gcp.zip](https://github.com/OracleFinder/CryptoCompareExternalAdapter/releases/download/v1.0/cl-cc-aws-gcp.zip)

### Docker
```bash
docker build . -t cryptocompareadaptor
docker run -d \
    --name cryptocompareadaptor \
    -p 80:80 \
    -e PORT=80 \
    cryptocompareadaptor
```


## Upload

Create a cloud function in GCP or Lambda, upload the zip file and set the handler function according to the platform you are using.

* GCP: `gcpservice`
* AWS: `handler`

## Test Cases (GCP/AWS test events)

### Fail

Event: 
```json
{
  "id": "278c97ffadb54a5bbb93cfec5f7b5503",
  "data": {}
}
```

Result:
```json
{
  "jobRunID": "278c97ffadb54a5bbb93cfec5f7b5503",
  "data": {
    "Response": "Error",
    "Message": "",
    "Type": 1,
    "Aggregated": false,
    "Data": [],
    "Path": "/data/",
    "ErrorsSummary": "Not implemented"
  }
}
```

### Pass

Event:
```json
{
  "id": "278c97ffadb54a5bbb93cfec5f7b5503",
  "data": {
    "endpoint": "price",
    "fsym": "ETH",
    "tsyms": "USD"
  }
}
```

Result:
```json
{
  "jobRunID": "278c97ffadb54a5bbb93cfec5f7b5503",
  "data": {
    "USD": 285.58
  }
}
```

Event:
```json
{
  "id": "278c97ffadb54a5bbb93cfec5f7b5503",
  "data": {
    "endpoint": "price",
    "fsym": "ETH",
    "tsyms": "USD,EUR,JPY"
  }
}
```

Event:
```json
{
  "id": "278c97ffadb54a5bbb93cfec5f7b5503",
  "data": {
    "endpoint": "pricemulti",
    "fsyms": "BTC,ETH",
    "tsyms": "USD,EUR"
  }
}
```

Event:
```json
{
  "id": "278c97ffadb54a5bbb93cfec5f7b5503",
  "data": {
    "endpoint": "pricemultifull",
    "fsyms": "BTC,ETH",
    "tsyms": "USD,EUR"
  }
}
```

Event:
```json
{
  "id": "278c97ffadb54a5bbb93cfec5f7b5503",
  "data": {
    "endpoint": "generateAvg",
    "fsym": "ETH",
    "tsym": "USD",
    "exchange": "Kraken"
  }
}
```
