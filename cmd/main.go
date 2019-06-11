// Copyright 2019 Microsoft Corp.

package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"strings"
	"time"
	"path/filepath"

	"github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
	"microsoft.com/github-events-spider/github"
	"microsoft.com/github-events-spider/utils"

	ct "github.com/daviddengcn/go-colortext"
	"golang.org/x/oauth2"
)

var (
	globalContext context.Context
	globalCancel  context.CancelFunc

	threadsArg  		  int
	hostArg     		  string
	portArg     		  int
	userArg     		  string
	passwordArg 		  string
	sslArg      		  string
	dbNameArg   		  string
	dropDataArg 		  bool
	verboseArg  		  bool
	tokenArg    		  string
	backendsArg 		  int
	backendNamePatternArg string
	dataDirArg            string
)

func main() {
	globalContext, globalCancel = context.WithCancel(context.Background())

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	closeDone := make(chan struct{}, 1)
	go func() {
		sig := <-sc
		fmt.Printf("\nGot signal [%v] to exit.\n", sig)
		globalCancel()

		select {
		case <-sc:
			// send signal again, return directly
			fmt.Printf("\nGot signal [%v] again to exit.\n", sig)
			os.Exit(1)
		case <-time.After(30 * time.Second):
			fmt.Print("\nWait 30s for closed, force exit\n")
			os.Exit(1)
		case <-closeDone:
			return
		}
	}()

	rootCmd := &cobra.Command{
		Use:   "github-events-spider",
		Short: "Polling GitHub Events to Spider (MariaDB storage engine)",
		Args:  cobra.MinimumNArgs(0),
		Run:   runCommandFunc,
	}
	initClientCommand(rootCmd)

	cobra.EnablePrefixMatching = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}

	globalCancel()

	closeDone <- struct{}{}
}

func initClientCommand(m *cobra.Command) {
	m.Flags().IntVarP(&threadsArg, "threads", "t", 500, "number of threads used to insert data to database")
	m.Flags().StringVarP(&hostArg, "host", "H", "", "frontend MySQL Server (Spider node) Hostname")
	m.Flags().IntVarP(&portArg, "port", "P", 3306, "MySQL Server Port")
	m.Flags().StringVarP(&userArg, "user", "u", "", "MySQL Server Username")
	m.Flags().StringVarP(&passwordArg, "password", "p", "", "MySQL Server Password")
	m.Flags().StringVar(&sslArg, "ssl", "false", "MySQL Server SSL")
	m.Flags().StringVarP(&dbNameArg, "database", "d", "github", "database name")
	m.Flags().BoolVarP(&dropDataArg, "drop-data", "D", false, "drop the database and tables")
	m.Flags().BoolVarP(&verboseArg, "verbose", "v", false, "output detail information")
	m.Flags().StringVarP(&tokenArg, "access-token", "T", "", "GitHub API access token")
	m.Flags().IntVarP(&backendsArg, "backends", "b", 3, "number of backend database nodes")
	m.Flags().StringVarP(&backendNamePatternArg, "backend-name-pattern", "B", "spider-backend-%d", "pattern of backend MySQL Servers (data node) name")
	m.Flags().StringVar(&dataDirArg, "data-dir", "data", "directory to store the github events archives")
	m.MarkFlagRequired("host")
	m.MarkFlagRequired("user")
	m.MarkFlagRequired("password")
}

func generateFrontendDSN() string {
	cfg := mysql.NewConfig()
	cfg.User = userArg
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(hostArg, fmt.Sprintf("%v", portArg))
	cfg.Passwd = passwordArg
	cfg.DBName = "" // no database preselected for creating database if not exist
	cfg.TLSConfig = sslArg
	cfg.InterpolateParams = true
	dsn := cfg.FormatDSN()

	fmt.Printf("frontend dsn=%s\n", dsn)
	return dsn
}

func generateBackendDSN(idx int) string {
	cfg := mysql.NewConfig()
	cfg.User = userArg + fmt.Sprintf("@" + backendNamePatternArg, idx)
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(fmt.Sprintf(backendNamePatternArg + ".mariadb.database.azure.com", idx), fmt.Sprintf("%v", portArg))
	cfg.Passwd = passwordArg
	cfg.DBName = "" // no database preselected for creating database if not exist
	cfg.TLSConfig = sslArg
	cfg.InterpolateParams = true
	dsn := cfg.FormatDSN()

	fmt.Printf("backend %d dsn=%s\n", idx, dsn)
	return dsn
}

func runCommandFunc(cmd *cobra.Command, args []string) {
	dsn := generateFrontendDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("Open DB failed: %s", err)
	}
	defer db.Close()
	db.SetMaxIdleConns(threadsArg + 1)
	db.SetMaxOpenConns(threadsArg * 2)

	var backendDBs []*sql.DB
	for i := 1; i <= backendsArg; i++ {
		dsn := generateBackendDSN(i)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Printf("Open backend DB %d failed: %s", i, err)
		}
		defer db.Close()
		db.SetMaxIdleConns(1)
		db.SetMaxOpenConns(1)

		backendDBs = append(backendDBs, db)
	}

	if err := createDatabase(globalContext, db, backendDBs); err != nil {
		log.Fatalf("Create database failed: %v", err)
	}
	if err := createTables(globalContext, db, backendDBs); err != nil {
		log.Fatalf("Create tables failed: %v", err)
	}
	if err := createIndexes(globalContext, db); err != nil {
		log.Fatalf("Create indexes failed: %v", err)
	}
	if err := createProcedures(globalContext, backendDBs); err != nil {
		log.Fatalf("Create procedure failed: %v", err)
	}

	// For bulk-loading historical GitHub events
	go bulkLoadGitHubEventArchives(globalContext, db)
	go invokeBackendProcedures(globalContext, backendDBs)

	// For polling real-time GitHub events
	// eventsCh := make(chan *github.Event, 100000)
	// go pollGitHubEvents(globalContext, eventsCh)
	// go insertGitHubEvents(globalContext, eventsCh, db)

	<-globalContext.Done()
}

func invokeBackendProcedures(ctx context.Context, backendDBs []*sql.DB) {
	for {
		commitsCnt := int64(0)
		
		startTime := time.Now()

		var wg sync.WaitGroup
		wg.Add(backendsArg)
		for _, backendDB := range backendDBs {
			go func(db *sql.DB) {
				defer wg.Done()
		
				cnt := 0
				if err := db.QueryRowContext(ctx, fmt.Sprintf("CALL %s.extract_commits();", dbNameArg)).Scan(&cnt); err != nil {
					log.Printf("%s", err)
					return
				}

				atomic.AddInt64(&commitsCnt, int64(cnt))
			}(backendDB)
		}
		wg.Wait()

		endTime := time.Now()

		ct.Foreground(ct.Green, true)
		log.Printf("extract_commits() processed %d events in %.2f seconds", atomic.LoadInt64(&commitsCnt), endTime.Sub(startTime).Seconds())
		ct.ResetColor()

		select {
		case <-ctx.Done():
			return
		default:
		}

		time.Sleep(20000 * time.Millisecond)
	}
}

func insertGitHubEvents(ctx context.Context, eventsCh chan *github.Event, db *sql.DB, finished chan bool) {
	sql := fmt.Sprintf("INSERT IGNORE INTO %s.github_events VALUES(?, ?, ?);", dbNameArg)
	stmt, err := db.PrepareContext(ctx, sql)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var wg sync.WaitGroup
	wg.Add(threadsArg)
	for i := 1; i <= threadsArg; i++ {
		go func(threadID int) {
			defer wg.Done()

			for {
				event := <-eventsCh
				if event == nil {
					return
				}

				data, err := json.Marshal(event)
				if err != nil {
					log.Println(err)
				}

				if _, err := stmt.ExecContext(ctx, event.GetID(), event.GetRepo().GetID(), string(data)); err != nil {
					log.Printf("%s", err)
					continue
				}
				
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}(i)
	}
	wg.Wait()

	finished <- true
}

func bulkLoadGitHubEventArchives(ctx context.Context, db *sql.DB) {
	for year := 2019; year <= 2019; year++ {
		for month := 1; month <= 12; month++ {
			for day := 1; day <= 31; day++ {
				eventsCh := make(chan *github.Event, 100000)
				finishedCh := make(chan bool)
				cntRows := int64(0)
				cntBytes := int64(0)

				startTime := time.Now()

				go insertGitHubEvents(ctx, eventsCh, db, finishedCh)

				for hour := 0; hour <= 23; hour++ {
					date := fmt.Sprintf("%d-%02d-%02d-%d", year, month, day, hour)
					t, err := time.Parse("2006-01-02-15", date)
					if err != nil || t.After(time.Now()) {
						break
					}

					filename := fmt.Sprintf("%d-%02d-%02d-%d.json.gz", year, month, day, hour)
					filePath := filepath.Join(dataDirArg, filename)
					fileURL := "https://data.gharchive.org/" + filename
				
					if _, err := os.Stat(filePath); os.IsNotExist(err) {
						if err := utils.DownloadFile(filePath, fileURL); err != nil {
							log.Printf("Failed to download %s to %s: %v", fileURL, filePath, err)
							continue
						}
					}

					if b, err := utils.ReadGzFile(filePath); err != nil {
						log.Printf("Failed to read %s: %v", filePath, err)
					} else {
						cntBytes += int64(len(b))  // use the uncompressed size

						scanner := bufio.NewScanner(bytes.NewReader(b))
						for scanner.Scan() {
							line := scanner.Text()
				
							var event *github.Event
							if err := json.NewDecoder(strings.NewReader(line)).Decode(&event); err != nil {
								log.Printf("Failed to parse %s to Event: %v", line, err)
							}
				
							eventsCh <- event
							cntRows++
						}
					}
				
					select {
					case <-ctx.Done():
						return
					default:
					}
				}

				for i := 1; i <= threadsArg; i++ {
					eventsCh <- nil
				}

				<-finishedCh
				endTime := time.Now()
				
				ct.Foreground(ct.Cyan, true)
				log.Printf("Ingested %d GitHub events (%.2f GB/minute)", cntRows, float64(cntBytes) / 1024 / 1024 / 1024 / endTime.Sub(startTime).Minutes())
				ct.ResetColor()
			}
		}
	}
}

func pollGitHubEvents(ctx context.Context, eventsCh chan *github.Event) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: tokenArg},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	opt := &github.ListOptions{Page: 1, PerPage: 100}
	etag := ""

	t := time.NewTicker(time.Duration(800) * time.Millisecond) // rate limit: 5000
	defer t.Stop()

	for {
		select {
		case <-t.C:
			go func() {
				events, resp, err := client.Activity.ListEventsWithETag(ctx, opt, etag)
				if err != nil && resp != nil && resp.StatusCode != 304 {
					log.Println(err)
				}

				if newETag := resp.Header.Get("ETag"); newETag != "" {
					etag = newETag
				}

				for _, event := range events {
					eventsCh <- event
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}

func createDatabase(ctx context.Context, db *sql.DB, backendDBs []*sql.DB) error {
	if dropDataArg {
		sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbNameArg)
		if _, err := execSQL(ctx, db, sql); err != nil {
			return err
		}
		for _, backendDB := range backendDBs {
			if _, err := execSQL(ctx, backendDB, sql); err != nil {
				return err
			}
		}
	}

	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", dbNameArg)
	_, err := execSQL(ctx, db, sql)
	for _, backendDB := range backendDBs {
		if _, err := execSQL(ctx, backendDB, sql); err != nil {
			return err
		}
	}

	return err
}

func createTables(ctx context.Context, db *sql.DB, backendDBs []*sql.DB) error {
	partitions := ""
	for i := 1; i <= backendsArg; i++ {
		if i > 1 {
			partitions += ",\n"
		}
		partitions += fmt.Sprintf("PARTITION pt%d COMMENT = 'srv \"backend%d\"'", i, i)
	}

	// github_events
	if err := createTable(ctx, db, "github_events",
		`event_id BIGINT NOT NULL PRIMARY KEY,
	 	 repo_id BIGINT NOT NULL,
		 data JSON NOT NULL`,
		`ENGINE=spider COMMENT='wrapper "mysql", table "github_events"'
		 PARTITION BY KEY (event_id)
		 (
			` + partitions + `
		 )`); err != nil {
		return err
	}

	// github_commits
	if err := createTable(ctx, db, "github_commits",
		`event_id BIGINT,
		 repo_id BIGINT,
		 repo_name VARCHAR(100),
		 pusher_login TEXT,
		 branch TEXT,
		 created_at TIMESTAMP,
		 author_name TEXT,
		 sha VARCHAR(100),
		 message TEXT,
		 commit TEXT`,
		`ENGINE=spider COMMENT='wrapper "mysql", table "github_commits"'
		 PARTITION BY KEY (event_id)
		 (
		   ` + partitions + `
		 )`); err != nil {
		return err
	}

	for _, backendDB := range backendDBs {
		// github_events
		if err := createTable(ctx, backendDB, "github_events",
			`event_id BIGINT NOT NULL PRIMARY KEY,
			 repo_id BIGINT NOT NULL,
			 data JSON NOT NULL`, ""); err != nil {
			return err
		}

		// github_commits
		if err := createTable(ctx, backendDB, "github_commits",
			`event_id BIGINT,
			 repo_id BIGINT,
			 repo_name VARCHAR(100),
			 pusher_login TEXT,
			 branch TEXT,
			 created_at TIMESTAMP,
			 author_name TEXT,
			 sha VARCHAR(100),
			 message TEXT,
			 commit TEXT`, ""); err != nil {
			return err
		}

		// daily_github_commits
		if err := createTable(ctx, backendDB, "daily_github_commits",
			`repo_id BIGINT,
			 repo_name VARCHAR(100),
			 day DATE,
			 num_commits BIGINT,
			 PRIMARY KEY (repo_id, day)`, ""); err != nil {
			return err
		}

		// github_commits_rollup
		if err := createTable(ctx, backendDB, "github_commits_rollup",
			`last_aggregated_id BIGINT`, ""); err != nil {
			return err
		}
		if dropDataArg {
			if _, err := execSQL(ctx, backendDB, fmt.Sprintf("INSERT INTO %s.github_commits_rollup VALUES (0);", dbNameArg)); err != nil {
				return err
			}
		}

		// numbers
		if err := createTable(ctx, backendDB, "numbers",
			`n INT PRIMARY KEY`, ""); err != nil {
			return err
		}
		if dropDataArg {
			for i := 0; i <= 50; i++ {  // allowed at most 50 commits in one push event
				if _, err := execSQL(ctx, backendDB, fmt.Sprintf("INSERT INTO %s.numbers VALUES (%d);", dbNameArg, i)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func createTable(ctx context.Context, db *sql.DB, tableName string, columns string, options string) error {
	if dropDataArg {
		sql := fmt.Sprintf("DROP TABLE IF EXISTS %s.%s;", dbNameArg, tableName)
		if _, err := execSQL(ctx, db, sql); err != nil {
			return err
		}
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (%s) %s;", dbNameArg, tableName, columns, options)
	_, err := execSQL(ctx, db, sql)
	return err
}

func createIndexes(ctx context.Context, db *sql.DB) error {
	if err := createIndex(ctx, db, "github_commits", "event_id", "commit_event_id_idx"); err != nil {
		return err
	}
	if err := createIndex(ctx, db, "github_commits", "repo_name", "commit_repo_name_idx"); err != nil {
		return err
	}
	if err := createIndex(ctx, db, "github_commits", "repo_id", "commit_repo_id_idx"); err != nil {
		return err
	}
	if err := createIndex(ctx, db, "github_commits", "sha", "commit_sha_idx"); err != nil {
		return err
	}
	if err := createIndex(ctx, db, "github_commits", "created_at", "commit_created_at_idx"); err != nil {
		return err
	}
	return nil
}

func createIndex(ctx context.Context, db *sql.DB, tableName string, columns string, indexName string) error {
	if dropDataArg {
		sql := fmt.Sprintf("DROP INDEX IF EXISTS %s ON %s.%s;", indexName, dbNameArg, tableName)
		if _, err := execSQL(ctx, db, sql); err != nil {
			return err
		}
	}

	sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s.%s (%s);", indexName, dbNameArg, tableName, columns)
	_, err := execSQL(ctx, db, sql)
	return err
}

func createProcedures(ctx context.Context, backendDBs []*sql.DB) error {
	for _, backendDB := range backendDBs {
		if err := createProcedure(ctx, backendDB, "extract_commits", "", 
			`DECLARE start_id BIGINT;
			 DECLARE end_id BIGINT;
 
			 SELECT last_aggregated_id+1
			 INTO start_id
			 FROM github_commits_rollup
			 FOR UPDATE;
 
			 SELECT MAX(event_id) 
			 INTO end_id
			 from github_events;
 
			 IF start_id <= end_id THEN 
			 	 INSERT IGNORE INTO
			 	 	 github_commits (event_id, repo_id, repo_name, pusher_login, branch, created_at, author_name, sha, message)
			 	 SELECT
			 	 	 event_id,
			 	 	 repo_id,
			 	 	 repo_name,
			 	 	 actor_login,
			 	 	 branch,
			 	 	 created_at,
			 	 	 JSON_UNQUOTE(JSON_EXTRACT(cmt, "$.author.name")) author_name,
			 	 	 JSON_UNQUOTE(JSON_EXTRACT(cmt, "$.sha")) sha,
			 	 	 JSON_UNQUOTE(JSON_EXTRACT(cmt, "$.message")) message
			 	 FROM (
			 	 	 SELECT
			 	 	 	 event_id,
			 	 	 	 repo_id,
			 	 	 	 repo_name,
			 	 	 	 actor_login,
			 	 	 	 JSON_UNQUOTE(JSON_EXTRACT(payload, "$.ref")) branch,
			 	 	 	 created_at,
			 	 	 	 JSON_UNQUOTE(JSON_EXTRACT(payload, CONCAT("$.commits[", numbers.n, "]"))) cmt
			 	 	 FROM numbers INNER JOIN (
			 	 	 	 SELECT
			 	 	 	 	 event_id,
			 	 	 	 	 repo_id,
			 	 	 	 	 JSON_UNQUOTE(JSON_EXTRACT(data, "$.repo.name")) repo_name,
			 	 	 	 	 JSON_UNQUOTE(JSON_EXTRACT(data, "$.created_at")) created_at,
			 	 	 	 	 JSON_UNQUOTE(JSON_EXTRACT(data, "$.actor.login")) actor_login,
			 	 	 	 	 JSON_UNQUOTE(JSON_EXTRACT(data, "$.payload")) payload
			 	 	 	 FROM 
			 	 	 	 	 github_events
			 	 	 	 WHERE
			 	 	 	 	 JSON_EXTRACT(data, "$.type") = "PushEvent" AND event_id BETWEEN start_id AND end_id
			 	 	 ) events ON numbers.n < JSON_LENGTH(payload, "$.commits")
			 	 ) commits;
	 
			 	 INSERT INTO
			 	 	 daily_github_commits
			 	 SELECT
			 	 	 repo_id,
			 	 	 repo_name,
			 	 	 DATE_FORMAT(created_at, '%Y-%m-%d'),
			 	 	 count(*)
			 	 FROM
			 	 	 github_commits
			 	 WHERE
			 	 	 event_id BETWEEN start_id AND end_id AND branch = 'refs/heads/master'
			 	 GROUP BY
			 	 	 1, 3
			 	 ON DUPLICATE KEY UPDATE
			 	 	 num_commits = num_commits + VALUES(num_commits);
		 
			 	 UPDATE github_commits_rollup SET last_aggregated_id = end_id;
			 END IF;

			 SELECT count(*)
			 FROM github_commits
			 WHERE event_id BETWEEN start_id AND end_id;`); err != nil {
			return err
		}
	}
	return nil
}

func createProcedure(ctx context.Context, db *sql.DB, procName string, procParam string, procBody string) error {
	if dropDataArg {
		sql := fmt.Sprintf("DROP PROCEDURE IF EXISTS %s.%s;", dbNameArg, procName)
		if _, err := execSQL(ctx, db, sql); err != nil {
			return err
		}
	}

	sql := fmt.Sprintf(`CREATE OR REPLACE PROCEDURE %s.%s(%s)
						BEGIN
							%s
						END;`, dbNameArg, procName, procParam, procBody)
	_, err := execSQL(ctx, db, sql)
	return err
}

func execSQL(ctx context.Context, db *sql.DB, sql string) (sql.Result, error) {
	if verboseArg {
		log.Printf("%v: %s", db, sql)
	}
	res, err := db.ExecContext(ctx, sql)
	if verboseArg {
		if err != nil {
			log.Printf("%v: %v", db, err)
		}
	}
	return res, err
}
