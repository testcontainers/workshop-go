# Step 2: Exploring the app

The app is a simple microservice for rating conference talks. It provides a web API to track the ratings of the talks in real time, storing the results in a PostgreSQL database and a Redis cache, and using Redpanda as a broker for the event stream.

## Storage layer

### SQL database with the talks

When a rating is submitted, we must verify that the talk for the given ID is present in our database.

Our database of choice is PostgreSQL, accessed with [jackc/pgx](https://github.com/jackc/pgx) PostgreSQL Driver.

Check `internal/talks/repo.go`.

### Redis

We store the ratings in Redis database with [redis/go-redis](https://github.com/redis/go-redis) Redis client.

Check `internal/ratings/repo.go`.

### Redpanda

We use ES/CQRS to materialize the events into the state. Redpanda acts as a broker and we use the Testcontainers' Redpanda module.

Check `internal/streams/broker.go`.

## Web application

### API

The API is a [Gin](https://gin-gonic.com) REST controller and exposes two endpoints:

* `POST /ratings { "talkId": ?, "value": 1-5 }` to add a rating for a talk
* `GET /ratings?talkId=?` to get the histogram of ratings of the given talk

Check `internal/app/handlers.go`.

### 
[Next](step-3-running-the-app-locally.md)
