# Pattern Implement

This tool generates implementation for given code(interface, function definition, etc.), for example when you want to implement caching wrapper for given interface you can pass your interface to `go-pattern-implement` and get implementation.

```
> cat inputs/cache

type Repo interface {
	Get(context.Context, string) (User, error)
}
```

```
> cat inputs/cache | go-pattern-implement implement cache --package user

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
go install github.com/relardev/go-pattern-implement@latest
```

## Integration

1. NeoVim - [link](https://github.com/relardev/go-pattern-implement.nvim)

## Usage

List all implementations

```
go-pattern-implement list
```

List only available implementations for given input

```
cat inputs/prometheus | go-pattern-implement list --available
```


Implement a pattern

```
cat inputs/prometheus | go-pattern-implement implement prometheus --package asdf
```

## Patterns

- [x] Metrics
    -  Prometheus
    -  StatsD
- [x] Tracing
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
- [ ] Paralellisation
- [ ] Batching
- [x] Throttle
    -  Error
    -  No error
- [ ] Retry
- [ ] Renew (for example token)

## TODOs

- [ ] Auto run tests on commits
