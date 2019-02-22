# od-database Go crawler 🚀
[![Build Status](https://travis-ci.org/terorie/od-database-crawler.svg?branch=master)](https://travis-ci.org/terorie/od-database-crawler)
> by terorie 2018 :P

 * Crawler for [__OD-Database__](https://github.com/simon987/od-database)
 * Crawls HTTP open directories (standard Web Server Listings)
 * Gets name, path, size and modification time of all files
 * Lightweight and fast: __over 9000 requests per second__ on a standard laptop

https://od-db.the-eye.eu/

## Usage

### Deploys

 1. With Config File (if `config.yml` found in working dir)
    - Download [default config](https://github.com/terorie/od-database-crawler/blob/master/config.yml)
    - Set `server.url` and `server.token`
    - Start with `./od-database-crawler server --config <file>`

 2. With Flags or env
    - Override config file if it exists
    - `--help` for list of flags
    - Every flag is available as an environment variable:
      `--server.crawl_stats` ➡️ `OD_SERVER_CRAWL_STATS`
    - Start with `./od-database-crawler server <flags>`

 3. With Docker
    ```dockerfile
    docker run \
        -e OD_SERVER_URL=xxx \
        -e OD_SERVER_TOKEN=xxx \
        terorie/od-database-crawler
    ```

### Flag reference

Here are the most important config flags. For more fine control, take a look at `/config.yml`.

| Flag/Config             | Environment/Docker         | Description                                                  | Example                             |
| ----------------------- | -------------------------- | ------------------------------------------------------------ | ----------------------------------- |
| `server.url`            | `OD_SERVER_URL`            | OD-DB Server URL                                             | `https://od-db.mine.the-eye.eu/api` |
| `server.token`          | `OD_SERVER_TOKEN`          | OD-DB Server Access Token                                    | _Ask Hexa **TM**_                   |
| `server.recheck`        | `OD_SERVER_RECHECK`        | Job Fetching Interval                                        | `3s`                                |
| `output.crawl_stats`    | `OD_OUTPUT_CRAWL_STATS`    | Crawl Stats Logging Interval (0 = disabled)                  | `500ms`                             |
| `output.resource_stats` | `OD_OUTPUT_RESORUCE_STATS` | Resource Stats Logging Interval (0 = disabled)               | `8s`                                |
| `output.log`            | `OD_OUTPUT_LOG`            | Log File (none = disabled)                                   | `crawler.log`                       |
| `crawl.tasks`           | `OD_CRAWL_TASKS`           | Max number of sites to crawl concurrently                    | `500`                               |
| `crawl.connections`     | `OD_CRAWL_CONNECTIONS`     | HTTP connections per site                                    | `1`                                 |
| `crawl.retries`         | `OD_CRAWL_RETRIES`         | How often to retry after a temporary failure (e.g. `HTTP 429` or timeouts) | `5`                                 |
| `crawl.dial_timeout`    | `OD_CRAWL_DIAL_TIMEOUT`    | TCP Connect timeout                                          | `5s`                                |
| `crawl.timeout`         | `OD_CRAWL_TIMEOUT`         | HTTP request timeout                                         | `20s`                               |
| `crawl.user-agent`      | `OD_CRAWL_USER_AGENT`      | HTTP Crawler User-Agent                                      | `googlebot/1.2.3`                   |
| `crawl.job_buffer`      | `OD_CRAWL_JOB_BUFFER`      | Number of URLs to keep in memory/cache, per job. The rest is offloaded to disk. Decrease this value if the crawler uses too much RAM. (0 = Disable Cache, -1 = Only use Cache) | `5000`                              |


