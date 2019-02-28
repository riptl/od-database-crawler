# OD-Database Crawler 🕷
[![Build Status](https://travis-ci.org/terorie/od-database-crawler.svg?branch=master)](https://travis-ci.org/terorie/od-database-crawler)
[![](https://tokei.rs/b1/github/terorie/od-database-crawler)](https://github.com/terorie/od-database-crawler)
[![CodeFactor](https://www.codefactor.io/repository/github/terorie/od-database-crawler/badge/master)](https://www.codefactor.io/repository/github/terorie/od-database-crawler/overview/master)

 * Crawler for [__OD-Database__](https://github.com/simon987/od-database)
 * In production at https://od-db.the-eye.eu/
 * Over 880 TB actively crawled
 * Crawls HTTP open directories (standard Web Server Listings)
 * Gets name, path, size and modification time of all files
 * Lightweight and fast

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
    ```bash
    docker run \
        -e OD_SERVER_URL=xxx \
        -e OD_SERVER_TOKEN=xxx \
        terorie/od-database-crawler
    ```

### Flag reference

Here are the most important config flags. For more fine control, take a look at `/config.yml`.

| Flag/Environment                                        | Description                                                  | Example                             |
| ------------------------------------------------------- | ------------------------------------------------------------ | ----------------------------------- |
| `server.url`<br />`OD_SERVER_URL`                       | OD-DB Server URL                                             | `https://od-db.mine.the-eye.eu/api` |
| `server.token`<br />`OD_SERVER_TOKEN`                   | OD-DB Server Access Token                                    | _Ask Hexa **TM**_                   |
| `server.recheck`<br />`OD_SERVER_RECHECK`               | Job Fetching Interval                                        | `3s`                                |
| `output.crawl_stats`<br />`OD_OUTPUT_CRAWL_STATS`       | Crawl Stats Logging Interval (0 = disabled)                  | `500ms`                             |
| `output.resource_stats`<br />`OD_OUTPUT_RESORUCE_STATS` | Resource Stats Logging Interval (0 = disabled)               | `8s`                                |
| `output.log`<br />`OD_OUTPUT_LOG`                       | Log File (none = disabled)                                   | `crawler.log`                       |
| `crawl.tasks`<br />`OD_CRAWL_TASKS`                     | Max number of sites to crawl concurrently                    | `500`                               |
| `crawl.connections`<br />`OD_CRAWL_CONNECTIONS`         | HTTP connections per site                                    | `1`                                 |
| `crawl.retries`<br />`OD_CRAWL_RETRIES`                 | How often to retry after a temporary failure (e.g. `HTTP 429` or timeouts) | `5`                                 |
| `crawl.dial_timeout`<br />`OD_CRAWL_DIAL_TIMEOUT`       | TCP Connect timeout                                          | `5s`                                |
| `crawl.timeout`<br />`OD_CRAWL_TIMEOUT`                 | HTTP request timeout                                         | `20s`                               |
| `crawl.user-agent`<br />`OD_CRAWL_USER_AGENT`           | HTTP Crawler User-Agent                                      | `googlebot/1.2.3`                   |
| `crawl.job_buffer`<br />`OD_CRAWL_JOB_BUFFER`           | Number of URLs to keep in memory/cache, per job. The rest is offloaded to disk. Decrease this value if the crawler uses too much RAM. (0 = Disable Cache, -1 = Only use Cache) | `5000`                              |
