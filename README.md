# Matrix Search Filter

Matrix search filter is a middleware plugin for [Traefik](https://github.com/traefik/traefik) which rewrites the HTTP response body for matrix search requests.

## Configuration

### Static

```toml
[pilot]
  token = "xxxx"

[experimental.plugins.rewritebody]
  modulename = "github.com/Awesome-Technologies/traefik-plugin-matrixsearchfilter"
  version = "v0.1.0"
```

### Dynamic

To configure the `Matrix Search Filter` plugin you should create a [middleware](https://doc.traefik.io/traefik/middlewares/overview/) in
your dynamic configuration as explained [here](https://doc.traefik.io/traefik/middlewares/overview/).
The following example creates and uses the `matrixsearchfilter` middleware plugin to remove all external MXIDs in the HTTP response body.

If you want to apply some limits on the response body, you can chain this middleware plugin with the [Buffering middleware](https://doc.traefik.io/traefik/middlewares/http/buffering/) from Traefik.

```toml
[http.routers]
  [http.routers.my-router]
    rule = "Host(`localhost`)"
    middlewares = ["matrix-search-filter"]
    service = "my-service"

[http.middlewares]
  [http.middlewares.matrix-search-filter.plugin.matrixsearchfilter]
    # Keep Last-Modified header returned by the HTTP service.
    # By default, the Last-Modified header is removed.
    lastModified = true
    # Keep all MXIDs with domain part equal to example.com
    userIdRegex = "^@[a-z0-9\._=\-\/\+]+:example\.com$"

[http.services]
  [http.services.my-service]
    [http.services.my-service.loadBalancer]
      [[http.services.my-service.loadBalancer.servers]]
        url = "http://127.0.0.1"
```
