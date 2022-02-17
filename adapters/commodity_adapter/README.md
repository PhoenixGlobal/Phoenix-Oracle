# commodity price api
This is a REST API that provides an up-to-date information of world commodities.

## Tools
- [Gin framework](https://gin-gonic.com)
- [gRPC](https://grpc.io)
- [Git](https://git-scm.com)
- [Docker Engine](https://www.docker.com)
- [Swagger](https://swagger.io)

## API documentation
The API uses the industry standart Swagger tool for its documentation.
Swagger documentation file: <a href="https://github.com/chutommy/market-info/blob/master/docs/swagger.json">JSON</a>/<a href="https://github.com/chutommy/market-info/blob/master/docs/swagger.yaml">YAML</a>

## Usage
### GET `/commodity/{name}`
If the commodity is supported, the server returns the commodity's `name`, current `price`/`currency`/`weight_unit`, the price's change in `percentage` and `float`, and the time of the `last update`.

## Examples
### GET /commodity/{name}: `/commodity/coal`
```
{
    "Name": "coal",
    "Price": 49.3,
    "Currency": "USD",
    "Weight_unit": "ton",
    "ChangeP": -0.4,
    "ChangeN": -0.2,
    "LastUpdate": 1597363200
}
```
## Configuration
**Default:**
```yaml
---
api_port: 80
commodity_service_target: 'localhost:10501'
```
*api_port:* the port of the host on which the server is running
*commodity_service_target:* the "host:post" of the commodity microservice server
