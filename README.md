# GitHub Events -> Spider -> MariaDB/MySQL

## Build
```
go build -o github-events-spider cmd/main.go
```

## Usage
```
github-events-spider [flags]
```

**Flags:**

| Shortcut | Name | Type | Description | Default |
| --- | --- | --- | --- | --- |
|  -T | --access-token         | string | GitHub API access token                           |                    |
|  -B | --backend-name-pattern | string | pattern of backend MySQL Servers (data node) name | "spider-backend-%d |
|  -b | --backends             | int    | number of backend database nodes                  | 3                  |
|     | --data-dir             | string | directory to store the github events archives     | "data"             |
|  -d | --database             | string | database name                                     | "github"           |
|  -D | --drop-data            |        | drop the database and tables                      |                    |
|  -h | --help                 |        | help for github-events-spider                     |                    |
|  -H | --host                 | string | frontend MySQL Server (Spider node) Hostname      |                    |
|  -p | --password             | string | MySQL Server Password                             |                    |
|  -P | --port                 | int    | MySQL Server Port                                 | 3306               |
|     |  --ssl                 | string | MySQL Server SSL                                  | "false"            |
|  -t | --threads              | int    | number of threads used to insert data to database | 500                |
|  -u | --user                 | string | MySQL Server Username                             |                    |
|  -v | --verbose              |        | output detail information                         |                    |

**Required flags:** `host`, `password`, `user`

## Demo

**Environment:**
  - 1 * Spider Node
    - Azure VM + MariaDB 10.4 latest Spider Dev branch *bb-10.4-spider-ks*@5b749e6aacbede0e5657aaf74d878b8adf2d2d7c
    - 32 vCore(s), 128 GB memory
  - 3 * Data Nodes
    - Azure Database for MariaDB 10.2
    - General Purpose, 32 vCore(s), 2048 GB

**Results:**
```log
~/github-events-spider$ ./github-events-spider -H xxx -u xxx -p xxx -t 500 -b 3 -D
frontend dsn=xxx:xxx@tcp(xxx:3306)/?interpolateParams=true&tls=false
backend 1 dsn=xxx:xxx@tcp(xxx.mariadb.database.azure.com:3306)/?interpolateParams=true&tls=false
backend 2 dsn=xxx:xxx@tcp(xxx.mariadb.database.azure.com:3306)/?interpolateParams=true&tls=false
backend 3 dsn=xxx:xxx@tcp(xxx.mariadb.database.azure.com:3306)/?interpolateParams=true&tls=false
2019/06/11 10:54:25 Ingested 154579 GitHub events (3.94 GB/minute)
2019/06/11 10:54:30 extract_commits() processed 95369 events in 9.71 seconds
2019/06/11 10:54:52 Ingested 166601 GitHub events (7.26 GB/minute)
2019/06/11 10:55:07 extract_commits() processed 139539 events in 16.34 seconds
2019/06/11 10:55:19 Ingested 100581 GitHub events (8.54 GB/minute)
2019/06/11 10:55:41 extract_commits() processed 125571 events in 14.80 seconds
2019/06/11 10:55:48 Ingested 114165 GitHub events (7.67 GB/minute)
2019/06/11 10:56:17 extract_commits() processed 121107 events in 15.30 seconds
2019/06/11 10:56:31 Ingested 195887 GitHub events (2.96 GB/minute)
2019/06/11 10:56:50 extract_commits() processed 119584 events in 13.01 seconds
2019/06/11 10:56:56 Ingested 155477 GitHub events (4.96 GB/minute)
2019/06/11 10:57:19 Ingested 66413 GitHub events (10.99 GB/minute)
2019/06/11 10:57:22 extract_commits() processed 129685 events in 12.65 seconds
2019/06/11 10:57:47 Ingested 115295 GitHub events (9.59 GB/minute)
2019/06/11 10:57:54 extract_commits() processed 94954 events in 11.42 seconds
2019/06/11 10:58:23 Ingested 166305 GitHub events (7.58 GB/minute)
2019/06/11 10:58:30 extract_commits() processed 133941 events in 16.26 seconds
2019/06/11 10:58:50 Ingested 123132 GitHub events (10.17 GB/minute)
2019/06/11 10:59:06 extract_commits() processed 127222 events in 16.03 seconds
2019/06/11 10:59:17 Ingested 149450 GitHub events (9.51 GB/minute)
2019/06/11 10:59:37 Ingested 125322 GitHub events (6.85 GB/minute)
2019/06/11 10:59:49 extract_commits() processed 172710 events in 23.37 seconds
2019/06/11 10:59:55 Ingested 83047 GitHub events (7.74 GB/minute)
2019/06/11 11:00:22 Ingested 151942 GitHub events (9.51 GB/minute)
2019/06/11 11:00:38 extract_commits() processed 219779 events in 28.28 seconds
2019/06/11 11:00:50 Ingested 132753 GitHub events (9.16 GB/minute)
2019/06/11 11:01:21 Ingested 153292 GitHub events (8.58 GB/minute)
2019/06/11 11:01:22 extract_commits() processed 189514 events in 24.05 seconds
2019/06/11 11:01:51 Ingested 160373 GitHub events (9.20 GB/minute)
2019/06/11 11:02:07 extract_commits() processed 196018 events in 24.83 seconds
2019/06/11 11:02:18 Ingested 131172 GitHub events (9.18 GB/minute)
2019/06/11 11:02:41 Ingested 145132 GitHub events (5.67 GB/minute)
2019/06/11 11:02:52 extract_commits() processed 181440 events in 25.52 seconds
2019/06/11 11:03:13 Ingested 207012 GitHub events (3.91 GB/minute)
2019/06/11 11:03:41 Ingested 125291 GitHub events (8.19 GB/minute)
2019/06/11 11:03:42 extract_commits() processed 239685 events in 29.47 seconds
2019/06/11 11:04:08 Ingested 119861 GitHub events (9.87 GB/minute)
2019/06/11 11:04:26 extract_commits() processed 185176 events in 23.94 seconds
2019/06/11 11:04:39 Ingested 159201 GitHub events (8.83 GB/minute)
2019/06/11 11:05:09 Ingested 167614 GitHub events (8.76 GB/minute)
2019/06/11 11:05:11 extract_commits() processed 168221 events in 25.36 seconds
2019/06/11 11:05:35 Ingested 110773 GitHub events (9.44 GB/minute)
2019/06/11 11:05:53 Ingested 107976 GitHub events (7.51 GB/minute)
2019/06/11 11:05:59 extract_commits() processed 198164 events in 28.31 seconds
2019/06/11 11:06:21 Ingested 174580 GitHub events (4.97 GB/minute)
2019/06/11 11:06:48 extract_commits() processed 238435 events in 29.25 seconds
2019/06/11 11:06:49 Ingested 143895 GitHub events (9.52 GB/minute)
2019/06/11 11:07:18 Ingested 121278 GitHub events (9.34 GB/minute)
2019/06/11 11:07:34 extract_commits() processed 181845 events in 25.15 seconds
2019/06/11 11:07:49 Ingested 191909 GitHub events (8.85 GB/minute)
2019/06/11 11:08:20 extract_commits() processed 201009 events in 26.51 seconds
2019/06/11 11:08:23 Ingested 102264 GitHub events (8.07 GB/minute)
2019/06/11 11:08:47 Ingested 98715 GitHub events (10.44 GB/minute)
2019/06/11 11:09:02 extract_commits() processed 145895 events in 21.83 seconds
2019/06/11 11:09:18 Ingested 194950 GitHub events (4.27 GB/minute)
2019/06/11 11:09:44 Ingested 168108 GitHub events (4.72 GB/minute)
2019/06/11 11:09:47 extract_commits() processed 180983 events in 25.45 seconds
2019/06/11 11:10:23 Ingested 122243 GitHub events (6.42 GB/minute)
2019/06/11 11:10:35 extract_commits() processed 201471 events in 27.25 seconds
2019/06/11 11:10:50 Ingested 110683 GitHub events (10.06 GB/minute)
2019/06/11 11:11:12 extract_commits() processed 116657 events in 17.63 seconds
2019/06/11 11:11:20 Ingested 100494 GitHub events (9.68 GB/minute)
2019/06/11 11:11:51 extract_commits() processed 121827 events in 19.12 seconds
2019/06/11 11:12:05 Ingested 188763 GitHub events (6.57 GB/minute)
2019/06/11 11:12:31 extract_commits() processed 117542 events in 19.50 seconds
2019/06/11 11:12:42 Ingested 123425 GitHub events (7.81 GB/minute)
2019/06/11 11:13:11 Ingested 175836 GitHub events (5.77 GB/minute)
2019/06/11 11:13:15 extract_commits() processed 143930 events in 24.06 seconds
2019/06/11 11:13:43 Ingested 187048 GitHub events (5.22 GB/minute)
2019/06/11 11:14:10 extract_commits() processed 261552 events in 35.11 seconds
2019/06/11 11:14:15 Ingested 119173 GitHub events (9.66 GB/minute)
2019/06/11 11:14:58 Ingested 177556 GitHub events (8.14 GB/minute)
2019/06/11 11:14:59 extract_commits() processed 181935 events in 28.94 seconds
2019/06/11 11:15:31 Ingested 101181 GitHub events (10.26 GB/minute)
2019/06/11 11:15:47 extract_commits() processed 184111 events in 28.05 seconds
2019/06/11 11:16:06 Ingested 190829 GitHub events (9.20 GB/minute)
2019/06/11 11:16:35 extract_commits() processed 183363 events in 28.27 seconds
2019/06/11 11:16:44 Ingested 169551 GitHub events (7.99 GB/minute)
2019/06/11 11:17:07 Ingested 127662 GitHub events (7.33 GB/minute)
2019/06/11 11:17:25 extract_commits() processed 182681 events in 29.88 seconds
2019/06/11 11:17:43 Ingested 228472 GitHub events (4.60 GB/minute)
2019/06/11 11:18:16 Ingested 136361 GitHub events (9.83 GB/minute)
2019/06/11 11:18:17 extract_commits() processed 248050 events in 31.25 seconds
2019/06/11 11:18:47 Ingested 65138 GitHub events (11.20 GB/minute)
2019/06/11 11:18:58 extract_commits() processed 125819 events in 21.23 seconds
2019/06/11 11:19:22 Ingested 169005 GitHub events (9.46 GB/minute)
2019/06/11 11:19:44 extract_commits() processed 150232 events in 26.07 seconds
2019/06/11 11:19:58 Ingested 167398 GitHub events (9.61 GB/minute)
2019/06/11 11:20:35 extract_commits() processed 174070 events in 31.23 seconds
2019/06/11 11:20:38 Ingested 233482 GitHub events (7.84 GB/minute)
2019/06/11 11:21:01 Ingested 140850 GitHub events (7.23 GB/minute)
2019/06/11 11:21:29 Ingested 172918 GitHub events (6.16 GB/minute)
2019/06/11 11:21:34 extract_commits() processed 243220 events in 38.46 seconds
2019/06/11 11:22:08 Ingested 171515 GitHub events (8.74 GB/minute)
2019/06/11 11:22:30 extract_commits() processed 252139 events in 36.27 seconds
2019/06/11 11:22:44 Ingested 150375 GitHub events (9.91 GB/minute)
2019/06/11 11:23:16 Ingested 96158 GitHub events (10.64 GB/minute)
2019/06/11 11:23:19 extract_commits() processed 191633 events in 29.54 seconds
2019/06/11 11:23:55 Ingested 159144 GitHub events (8.59 GB/minute)
2019/06/11 11:24:05 extract_commits() processed 146644 events in 25.20 seconds
2019/06/11 11:24:26 Ingested 109555 GitHub events (10.50 GB/minute)
2019/06/11 11:24:49 Ingested 138538 GitHub events (7.78 GB/minute)
2019/06/11 11:24:49 extract_commits() processed 125824 events in 24.43 seconds
2019/06/11 11:25:21 Ingested 201355 GitHub events (5.42 GB/minute)
2019/06/11 11:25:41 extract_commits() processed 215006 events in 32.26 seconds
2019/06/11 11:25:52 Ingested 108076 GitHub events (10.61 GB/minute)
2019/06/11 11:26:26 Ingested 119643 GitHub events (10.25 GB/minute)
2019/06/11 11:26:29 extract_commits() processed 179992 events in 27.76 seconds
2019/06/11 11:27:07 Ingested 163360 GitHub events (8.39 GB/minute)
2019/06/11 11:27:18 extract_commits() processed 163178 events in 28.46 seconds
2019/06/11 11:27:39 Ingested 91730 GitHub events (10.81 GB/minute)
2019/06/11 11:28:00 extract_commits() processed 105828 events in 22.33 seconds
2019/06/11 11:28:11 Ingested 142245 GitHub events (9.70 GB/minute)
```
