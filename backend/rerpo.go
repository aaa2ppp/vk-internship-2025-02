package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

type repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) repo {
	return repo{db: db}
}

func (re repo) getLogger(ctx context.Context, op string) *slog.Logger {
	return GetLoggerFromContext(ctx).With("op", "repo."+op)
}

func (re repo) GetHosts(ctx context.Context) ([]Host, error) {
	log := re.getLogger(ctx, "GetHosts")

	const q = `SELECT host_id, host_name FROM host;`

	rows, err := re.db.QueryContext(ctx, q)
	if err != nil {
		log.Error(fmt.Sprintf("%v", err))
		return nil, errInternalError
	}
	defer rows.Close()

	hosts := []Host{}
	for rows.Next() {
		var (
			id   int
			name string
		)
		if err := rows.Scan(&id, &name); err != nil {
			log.Error(fmt.Sprintf("%v", err))
			return nil, errInternalError
		}
		hosts = append(hosts, Host{id, name})
	}

	if err := rows.Err(); err != nil {
		log.Error(fmt.Sprintf("%v", err))
		return nil, errInternalError
	}

	log.Debug("", "hosts", hosts)
	return hosts, nil
}

func (re repo) AddHosts(ctx context.Context, hosts []string) error {
	log := re.getLogger(ctx, "AddHosts")
	log.Debug("", "hosts", hosts)

	var q = `INSERT INTO host (host_name) VALUES (%s) ON CONFLICT DO NOTHING;`

	placeholders := make([]string, 0, len(hosts))
	values := make([]any, 0, len(hosts))

	for i := 0; i < len(hosts); i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		values = append(values, hosts[i])
	}

	q = fmt.Sprintf(q, strings.Join(placeholders, "),("))

	if _, err := re.db.ExecContext(ctx, q, values...); err != nil {
		log.Error(fmt.Sprintf("%v", err))
		return errInternalError
	}

	return nil
}

func (re repo) GetLastSuccessPingResults(ctx context.Context) ([]PingResult, error) {
	log := re.getLogger(ctx, "GetLastSuccessPingResults")

	// TODO: Поиск последней успешной записи в таблице ping_result этот неоптималеное
	// решение и может создавать нагрузку на базу из-за отсутствия индекса по
	// (host_id, ping_time). Создание индекса только для этого запроса нецелесообразно,
	// так как запрос выполняется всего один раз при старте приложения.
	// В качестве решения выбран вариант чтения ограниченного числа строк с конца лога
	// с последующей выборкой из них.
	// Альтернативное решение: поддержка отдельной таблицы с последними результатами для
	// каждого хоста.
	// Пока оставляем как есть, так как это макет.

	const logTailLimit = 1000 // TODO: to config
	const q = `WITH log_tail AS (
		SELECT *
		FROM ping_result
		ORDER BY id DESC
		LIMIT $1
	)
	SELECT
		h.host_id,
		h.host_name,
		lt.ip,
		lt.ping_time,
		lt.ping_rtt
	FROM (
		SELECT DISTINCT ON (host_id)
			host_id,
			ip,
			ping_time,
			ping_rtt
		FROM log_tail
		WHERE success
		ORDER BY host_id, ping_time DESC
	) AS lt
	JOIN host h USING (host_id)
	ORDER BY h.host_name;`

	rows, err := re.db.QueryContext(ctx, q, logTailLimit)
	if err != nil {
		log.Error(fmt.Sprintf("%v", err))
		return nil, errInternalError
	}
	defer rows.Close()

	results := []PingResult{}
	for rows.Next() {
		res := PingResult{Success: true}
		if err := rows.Scan(&res.HostID, &res.HostName, &res.IP, &res.Time, &res.Rtt); err != nil {
			log.Error(fmt.Sprintf("%v", err))
			return nil, errInternalError
		}
		results = append(results, res)
	}

	if err := rows.Err(); err != nil {
		log.Error(fmt.Sprintf("%v", err))
		return nil, errInternalError
	}

	log.Debug("", "results", results)
	return results, nil
}

func (re repo) AddPingResults(ctx context.Context, results []PingResult) error {
	log := re.getLogger(ctx, "AddPingResults")
	log.Debug("", "results", results)

	var q = `INSERT INTO ping_result (host_id, ip, ping_time, ping_rtt, success) VALUES (%s);`

	placeholders := make([]string, 0, len(results))
	values := make([]any, 0, len(results))

	for i, j := 0, 0; i < len(results); i, j = i+1, j+5 {
		p := &results[i]
		placeholders = append(placeholders, fmt.Sprintf("$%d,$%d,$%d,$%d,$%d", j+1, j+2, j+3, j+4, j+5))
		values = append(values, p.HostID, p.IP, p.Time, p.Rtt, p.Success)
	}

	q = fmt.Sprintf(q, strings.Join(placeholders, "),("))

	if _, err := re.db.ExecContext(ctx, q, values...); err != nil {
		log.Error(fmt.Sprintf("%v", err))
		return errInternalError
	}

	return nil
}
