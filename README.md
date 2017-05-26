# What is Locker?

Locker is a simple web application for claiming and releasing locks.
It contains a Concourse resource to help make locking various tasks
and jobs in Concourse pipelines easier to manage.

This project is similar to the [pool-resource](https://github.com/concourse/pool-resource), with a few key differences:

1) It uses a webserver with file-backed persistence for the locks. This isn't
   terribly scalable, but since it's handling locks for Concourse pipelines,
   it isn't anticipated to have an enormous traffic load.
2) Locks and Keys do not need to be pre-defined in order to be used
3) You may lock a Lock multiple times with the same Key. If this is done,
   you must unlock once for each lock in place on the key.
4) Each lock request may specify who it is, along with the key. If not specified,
   the value of the key will be used as the identifier. This allows users to lock
   a resource that can be shared by node-a, node-b, and node-c, but not node-e.

# How do I use it?

1) Deploy `locker` along side your Concourse database node via the [locker-boshrelease](https://github.com/cloudfoundry-community/locker-boshrelease)
2) Use the [locker-resource](https://github.com/cloudfoundry-community/locker-resource) in your Concourse pipelines.

# locker API

## Supported Requests

* `GET /locks`

  Returns a JSON formatted list of locks + who owns them currently

* `PUT /lock/<lock-name>`

  Content: `{"key":"key-to-lock-with", "lock_by": "identifier-of-lock-requestor"}`

  Issues a lock on `lock-name` keyed with `key` attribute in the JSON payload of the request.
  If the lock was already taken, it will immediately return a 423 error, and the client should back-off +
  re-try at a sane interval until the lock is obtained. The `lock_by` field is optional, and used to
  identify the item requesting the lock. If not specified, it will default to the value of `key`.

  `locked_by` allows you to lock a lock multiple times with a single key, by multiple items. Different
  keys should be specified by items that need exclusivity on the lock. If three things can run simultaneously,
  but the fourth item must be exclusive, use two keys, and one key should have three `locked_by` requestors.

  Returns 200 on success, 423 on locking failure

  Example to lock `prod-deployments` with `prod-cloudfoundry`:

  ```
  curl -X PUT -d '{"key":"prod-cloudfoundry"}' http://locker-ip:port/lock/prod-deployments
  # or to explicitly set the lock requestor
  curl -X PUT -d '{"key":"prod-cloudfoundry", "lock_by": "my-deployment"}' http://locker-ip:port/lock/prod-deployments
  ```

* `DELETE /lock/<lock-name>`

  Content: `{"key":"key-to-unlock-with"}`

  Issues an unlock request on `lock-name` using the `key` attribute in the JSON payload as the unlock key.
  If the lock on `lock-name` is not currently keyed by `key-to-unlock-with`, the
  unlock is disallowed. If the lock is currently not held by anyone, always returns 200.

  Returns 200 on success, 423 on failure.

  Example to unlock `prod-deployments` previously locked with `prod-cloudfoundry`:

  ```
  curl -X DELETE -d '{"key":"prod-cloudfoundry"}' http://locker-ip:port/lock/prod-deployments
  # or to unlock my-deployment's lock using `prod-cloudfoundry`
  curl -X DELETE -d '{"key":"prod-cloudfoundry","locked_by":"my-deployment"}' http://locker-ip:port/lock/prod-deployments
  ```

## Authentication

`locker` is protected with HTTP basic authentication. Credentials are passed in 

## Error handling

Errors will be reported as json objects, inside a top-level `error` key:

```
{"error":"This is what went wrong"}
```

## Running manually

If you want to run `locker` manually for testing/development:

```
go build
LOCKER_CONFIG=/tmp/locker-data.yml ./locker
```

`locker` can be configured using the following environment variables:

* `LOCKER_CONFIG` **required** - specifies the file that locks will be stored in
* `PORT` - Defaults to 3000, controls the port `locker` listens on
* `AUTH_USER` - If specified, requires `AUTH_PASS` and configures the username for
  HTTP basic auth
* `AUTH_PASS` - If specified, requires `AUTH_USER` and configures the password for
  HTTP basic auth
