# Exploring the running app

Once the application is running, you might want to connect to the running containers to inspect the data in the Postgres database or the elements in the Redis store.

With [Testcontainers Desktop](https://www.testcontainers.com/desktop), it's easy to do that.

To access the different services, please use your favorite client to connect to them and inspect the data. For the workshop, which was built with VSCode, we are using a [database client extension](https://doc.database-client.com/#/), as it allows connecting to different technologies.

## Connecting to the Database

From the root directory of the workshop, let's first start the application with `make dev`.

Now create a connection to the Postgres database. When you set `postgres` as user and password, and `talks-db` as database, the connection will fail with the following error:

> Connection error!connect ECONNREFUSED 127.0.0.1:5432

The well-known port for Postgres is 5432, but the connection is refused. Why?

By default, Testcontainers starts the containers and maps the ports to a random available port on the host. So, you need to find out the mapped port to connect to the database: put a break point, inspect the container, or check the logs, among other things.

Instead, we can use Testcontainers Desktop fixed ports support to connect to the database.

Open Testcontainers Desktop, and select the `Services` -> `Open config location`.
It will open a directory with the example configuration files for commonly used services.

Copy the `postgres.toml.example` to `postgres.toml`, and update it's content to the following:

```toml
ports = [
  {local-port = 5432, container-port = 5432},
]

selector.image-names = ["postgres"]
```

This configuration will map Postgres container port 5432 to the host port 5432. If you take a look at the Desktop UI, you will see that the Postgres service now appears in the services list, including additional sub-entries for tailing container logs and getting a shell into the container.

Now, let's try to create the connection again. This time, it will work.

And it will also work for the Redis store and the Redpanda streaming queue as well. Simply copy the `redis.toml.example` to `redis.toml`, and update it's content to the following:

```toml
ports = [
    { local-port = 6379, container-port = 6379 }
]

selector.image-names = ["redis"]
```

And finally add the `redpanda.toml`, and update it's content to the following:

```toml
ports = [
  {local-port = 9092, container-port = 9093},
]

selector.image-names = ["docker.redpanda.com/redpandadata/redpanda"]
```
