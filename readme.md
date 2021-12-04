## Miniblog web server

[OpenAPI specification.](api.yaml)

Run server:
```bash
$ go run server.go
```

Run server from docker container:
```bash
$ docker build . -t miniblog
$ docker run -e SERVER_PORT=8080 -p 8080:8080 miniblog
```

Run all containers (server, mongo, redis):
```
docker-compose up --build app
```

Environment variables:
- `SERVER_PORT` --- port number to run server on
- `STORAGE_MODE` --- storage mode, one of:
    - `inmemory` --- store data in memory
    - `mongo` --- store data in MongoDB. To use this mode, additional env vars must be specified:
      `MONGO_URL`, `MONGO_DBNAME`
    - `cached` --- store data in MongoDB with cache in Redis. To use this mode, additional env vars must be specified:
      `MONGO_URL`, `MONGO_DBNAME`, `REDIS_URL`
- `MONGO_URL` --- address to connect to MongoDB
- `MONGO_DBNAME` --- MongoDB database name
- `REDIS_URL` --- address to connect to Redis
- `APP_MODE` -- application mode. Possible values:
    - `SERVER` - server mode, accepts requests
    - `WORKER` - valid only for `STORAGE_MODE = mongo` configuration.
       Is used to update users' feeds in background using Redis broker.

