# Component Generator

This tool generates implementation for given code(interface, function definition, etc.), for example when you want to implement caching wrapper for given interface you can pass your interface to `go-component-generator` and get implementation.

```
> cat inputs/cache

type Repo interface {
	Get(context.Context, string) (User, error)
}
```

```
> cat inputs/cache | go-component-generator implement cache --package user

type Cache struct {
        r       user.Repo
        cache   *cache.Cache
}

func New(r user.Repo, expiration, cleanupInterval time.Duration) *Cache {
        return &Cache{r: r, cache: cache.New(expiration, cleanupInterval)}
}
func (r *Cache) Get(ctx context.Context, arg string) (user.User, error) {
        key := arg
        cachedItem, found := r.cache.Get(key)
        if found {
                user, ok := cachedItem.(user.User)
                if !ok {
                        return user.User{}, errors.New("invalid object in cache")
                }
                return user, nil
        }
        user := r.r.Get(ctx, arg)
        if err != nil {
                return user.User{}, err
        }
        r.cache.Set(key, user, cache.DefaultExpiration)
        return user, nil
}
```


## Install

```
go install github.com/relardev/go-component-generator
```

## Usage

List all implementations

```
go-component-generator list
```

List only available implementations for given input

```
cat inputs/prometheus | go-component-generator list --available
```


Implement a component

```
cat inputs/prometheus | go-component-generator implement prometheus --package asdf
```

## Components

- [x] Metrics
    -  Prometheus
    -  StatsD
- [ ] Tracing
- [x] Cache
- [x] Store
- [ ] Semaphore
    - [x] Basic
    - [ ] With Cancel
    - [ ] With Waitgroup
    - [ ] With Waitgroup and Cancel
- [x] File getter
- [x] Slog
- [ ] Log
- [x] Filter
    - Whole call
    - Result
    - Param
- [ ] Goroutine Channel Hinge
- [ ] Batching
- [x] Throttle
    -  Error
    -  No error
- [ ] Retry

## TODOs

- [ ] Auto run tests on commits
