package osquery

import (
	"net/url"
	"path"
)

// MDMlabFlags is the set of flags to pass to osquery when connecting to MDMlab.
func MDMlabFlags(mdmlabURL *url.URL) []string {
	hostname, prefix := mdmlabURL.Host, mdmlabURL.Path
	return []string{
		"--tls_hostname=" + hostname,
		"--enroll_tls_endpoint=" + path.Join(prefix, "/api/v1/osquery/enroll"),
		"--config_plugin=tls",
		"--config_tls_endpoint=" + path.Join(prefix, "/api/v1/osquery/config"),
		// Osquery defaults config_refresh to 0 which is probably not ideal for
		// a client connected to MDMlab. Users can always override this in the
		// config they serve via MDMlab.
		"--config_refresh=60",
		"--disable_distributed=false",
		"--distributed_plugin=tls",
		"--distributed_tls_max_attempts=10",
		"--distributed_tls_read_endpoint=" + path.Join(prefix, "/api/v1/osquery/distributed/read"),
		"--distributed_tls_write_endpoint=" + path.Join(prefix, "/api/v1/osquery/distributed/write"),
		"--logger_plugin=tls,filesystem",
		"--logger_tls_endpoint=" + path.Join(prefix, "/api/v1/osquery/log"),
		"--disable_carver=false",
		// carver_disable_function is separate from disable_carver as it controls the use of file
		// carving as a SQL function (eg. `SELECT carve(path) FROM processes`).
		"--carver_disable_function=false",
		"--carver_start_endpoint=" + path.Join(prefix, "/api/v1/osquery/carve/begin"),
		"--carver_continue_endpoint=" + path.Join(prefix, "/api/v1/osquery/carve/block"),
		"--carver_block_size=8000000",
	}
}
