# Architecture

`crux-console` is a client of `crux-server`.
It should not own fleet state, policy truth, approval truth, or trace storage.

```text
browser
    |
    v
crux-console
    |
    v
crux-server API
```

The console should expose the same operational model available through the CLI, with richer navigation and visualization.
