# Pool

[![Doc](https://pkg.go.dev/badge/github.com/bsm/pool)](https://pkg.go.dev/github.com/bsm/pool)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A simple connection pool for Go.

Features:

- Thread-safe (obviously)
- Stack based (rather than queue based) - connections that have been used recently are more likely to be re-used again
- Supports shirinking - idle pool connections can be reaped

## Credits

- https://github.com/PurpureGecko/go-lfc
- https://github.com/fatih/pool
- https://github.com/go-redis/redis
