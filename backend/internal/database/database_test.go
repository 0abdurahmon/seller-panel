package database

import "testing"

func TestToPgx5URL(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"no query": {
			"postgres://u:p@db:5432/seller",
			"pgx5://u:p@db:5432/seller",
		},
		"long scheme": {
			"postgresql://u:p@db/seller?sslmode=disable",
			"pgx5://u:p@db/seller?sslmode=disable",
		},
		"only pool parameters": {
			"postgres://u:p@db/seller?pool_max_conns=10&pool_min_conns=2",
			"pgx5://u:p@db/seller",
		},
		"pool parameters between others": {
			"postgres://u:p@db/seller?sslmode=disable&pool_max_conns=10&connect_timeout=5&pool_max_conn_lifetime=1h",
			"pgx5://u:p@db/seller?sslmode=disable&connect_timeout=5",
		},
		"percent encoded name": {
			"postgres://u:p@db/seller?pool%5Fmax_conns=10&sslmode=disable",
			"pgx5://u:p@db/seller?sslmode=disable",
		},
		"the rest keeps its encoding and order": {
			"postgres://u:p%40ss%3F@db/seller?search_path=a%2Cb&pool_max_conns=3&application_name=x+y&options=-c%20geqo%3Doff",
			"pgx5://u:p%40ss%3F@db/seller?search_path=a%2Cb&application_name=x+y&options=-c%20geqo%3Doff",
		},
		"pool in a value": {
			"postgres://u:p@db/pool_db?application_name=pool_worker",
			"pgx5://u:p@db/pool_db?application_name=pool_worker",
		},
		"another scheme": {
			"pgx5://u:p@db/seller?sslmode=disable",
			"pgx5://u:p@db/seller?sslmode=disable",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := toPgx5URL(c.in); got != c.want {
				t.Fatalf("toPgx5URL(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
