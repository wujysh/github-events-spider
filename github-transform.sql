DELIMITER //
CREATE OR REPLACE PROCEDURE github.extract_commits()
BEGIN
    DECLARE start_id BIGINT;
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
    WHERE event_id BETWEEN start_id AND end_id;
END//
DELIMITER ;
