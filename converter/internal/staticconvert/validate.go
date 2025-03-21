package staticconvert

import (
	"fmt"

	"github.com/grafana/agent/converter/diag"
	"github.com/grafana/agent/converter/internal/common"
	"github.com/grafana/agent/pkg/config"
	agent_exporter "github.com/grafana/agent/pkg/integrations/agent"
	"github.com/grafana/agent/pkg/integrations/apache_http"
	"github.com/grafana/agent/pkg/integrations/azure_exporter"
	"github.com/grafana/agent/pkg/integrations/blackbox_exporter"
	"github.com/grafana/agent/pkg/integrations/cadvisor"
	"github.com/grafana/agent/pkg/integrations/cloudwatch_exporter"
	"github.com/grafana/agent/pkg/integrations/consul_exporter"
	"github.com/grafana/agent/pkg/integrations/dnsmasq_exporter"
	"github.com/grafana/agent/pkg/integrations/elasticsearch_exporter"
	"github.com/grafana/agent/pkg/integrations/gcp_exporter"
	"github.com/grafana/agent/pkg/integrations/github_exporter"
	"github.com/grafana/agent/pkg/integrations/kafka_exporter"
	"github.com/grafana/agent/pkg/integrations/memcached_exporter"
	"github.com/grafana/agent/pkg/integrations/mongodb_exporter"
	mssql_exporter "github.com/grafana/agent/pkg/integrations/mssql"
	"github.com/grafana/agent/pkg/integrations/mysqld_exporter"
	"github.com/grafana/agent/pkg/integrations/node_exporter"
	"github.com/grafana/agent/pkg/integrations/oracledb_exporter"
	"github.com/grafana/agent/pkg/integrations/postgres_exporter"
	"github.com/grafana/agent/pkg/integrations/process_exporter"
	"github.com/grafana/agent/pkg/integrations/redis_exporter"
	"github.com/grafana/agent/pkg/integrations/snmp_exporter"
	"github.com/grafana/agent/pkg/integrations/snowflake_exporter"
	"github.com/grafana/agent/pkg/integrations/squid_exporter"
	"github.com/grafana/agent/pkg/integrations/statsd_exporter"
	"github.com/grafana/agent/pkg/integrations/windows_exporter"
	"github.com/grafana/agent/pkg/logs"
	"github.com/grafana/agent/pkg/metrics"
	"github.com/grafana/agent/pkg/server"
	"github.com/grafana/agent/pkg/traces"

	_ "github.com/prometheus/prometheus/discovery/install" // Register Prometheus SDs
)

func validate(staticConfig *config.Config) diag.Diagnostics {
	var diags diag.Diagnostics

	_, grpcListenPort, _ := staticConfig.ServerFlags.GRPC.ListenHostPort()

	diags.AddAll(validateCommandLine())
	diags.AddAll(validateServer(staticConfig.Server))
	diags.AddAll(validateMetrics(staticConfig.Metrics, grpcListenPort))
	diags.AddAll(validateIntegrations(staticConfig.Integrations))
	diags.AddAll(validateTraces(staticConfig.Traces))
	diags.AddAll(validateLogs(staticConfig.Logs))
	diags.AddAll(validateAgentManagement(staticConfig.AgentManagement))

	return diags
}

func validateCommandLine() diag.Diagnostics {
	var diags diag.Diagnostics

	diags.Add(diag.SeverityLevelWarn, "Please review your agent command line flags and ensure they are set in your Flow mode config file where necessary.")

	return diags
}

func validateServer(serverConfig *server.Config) diag.Diagnostics {
	var diags diag.Diagnostics

	defaultServerConfig := server.DefaultConfig()
	diags.AddAll(common.ValidateSupported(common.NotDeepEquals, serverConfig.GRPC, defaultServerConfig.GRPC, "grpc_tls_config server", "flow mode does not have a gRPC server to configure."))
	diags.AddAll(common.ValidateSupported(common.NotEquals, serverConfig.HTTP.TLSConfig.PreferServerCipherSuites, defaultServerConfig.HTTP.TLSConfig.PreferServerCipherSuites, "prefer_server_cipher_suites server", ""))

	return diags
}

// validateMetrics validates the metrics config for anything not already
// covered by appendStaticPrometheus.
func validateMetrics(metricsConfig metrics.Config, grpcListenPort int) diag.Diagnostics {
	var diags diag.Diagnostics

	defaultMetrics := config.DefaultConfig().Metrics
	defaultMetrics.ServiceConfig.Lifecycler.ListenPort = grpcListenPort
	diags.AddAll(common.ValidateSupported(common.NotDeepEquals, metricsConfig.WALCleanupAge, defaultMetrics.WALCleanupAge, "wal_cleanup_age metrics", ""))
	diags.AddAll(common.ValidateSupported(common.NotEquals, metricsConfig.WALCleanupPeriod, defaultMetrics.WALCleanupPeriod, "wal_cleanup_period metrics", ""))
	diags.AddAll(common.ValidateSupported(common.NotDeepEquals, metricsConfig.ServiceConfig, defaultMetrics.ServiceConfig, "scraping_service metrics", ""))
	diags.AddAll(common.ValidateSupported(common.NotDeepEquals, metricsConfig.ServiceClientConfig, defaultMetrics.ServiceClientConfig, "scraping_service_client metrics", ""))
	diags.AddAll(common.ValidateSupported(common.NotEquals, metricsConfig.InstanceRestartBackoff, defaultMetrics.InstanceRestartBackoff, "instance_restart_backoff metrics", ""))
	diags.AddAll(common.ValidateSupported(common.NotEquals, metricsConfig.InstanceMode, defaultMetrics.InstanceMode, "instance_mode metrics", ""))
	diags.AddAll(common.ValidateSupported(common.NotEquals, metricsConfig.DisableKeepAlives, defaultMetrics.DisableKeepAlives, "http_disable_keepalives metrics", ""))
	diags.AddAll(common.ValidateSupported(common.NotEquals, metricsConfig.IdleConnTimeout, defaultMetrics.IdleConnTimeout, "http_idle_conn_timeout metrics", ""))

	if metricsConfig.WALDir != defaultMetrics.WALDir {
		diags.Add(diag.SeverityLevelWarn, "The converter does not support converting the provided metrics wal_directory config: Use the run command flag --storage.path for Flow mode instead.")
	}

	return diags
}

func validateIntegrations(integrationsConfig config.VersionedIntegrations) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, integration := range integrationsConfig.ConfigV1.Integrations {
		if !integration.Common.Enabled {
			diags.Add(diag.SeverityLevelWarn, fmt.Sprintf("disabled integrations do nothing and are not included in the output: %s.", integration.Name()))
			continue
		}

		switch itg := integration.Config.(type) {
		case *agent_exporter.Config:
		case *apache_http.Config:
		case *node_exporter.Config:
		case *blackbox_exporter.Config:
		case *cloudwatch_exporter.Config:
		case *consul_exporter.Config:
		case *dnsmasq_exporter.Config:
		case *elasticsearch_exporter.Config:
		case *gcp_exporter.Config:
		case *github_exporter.Config:
		case *kafka_exporter.Config:
		case *memcached_exporter.Config:
		case *mongodb_exporter.Config:
		case *mssql_exporter.Config:
		case *mysqld_exporter.Config:
		case *oracledb_exporter.Config:
		case *postgres_exporter.Config:
		case *process_exporter.Config:
		case *redis_exporter.Config:
		case *snmp_exporter.Config:
		case *snowflake_exporter.Config:
		case *squid_exporter.Config:
		case *statsd_exporter.Config:
		case *windows_exporter.Config:
		case *azure_exporter.Config:
		case *cadvisor.Config:
		default:
			diags.Add(diag.SeverityLevelError, fmt.Sprintf("The converter does not support converting the provided %s integration.", itg.Name()))
		}
	}

	return diags
}

func validateTraces(tracesConfig traces.Config) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddAll(common.ValidateSupported(common.NotDeepEquals, tracesConfig, traces.Config{}, "traces", ""))

	return diags
}

// validateLogs validates the logs config for anything not already covered
// by appendStaticPromtail.
func validateLogs(logsConfig *logs.Config) diag.Diagnostics {
	var diags diag.Diagnostics

	return diags
}

func validateAgentManagement(agentManagementConfig config.AgentManagementConfig) diag.Diagnostics {
	var diags diag.Diagnostics

	diags.AddAll(common.ValidateSupported(common.NotDeepEquals, agentManagementConfig, config.AgentManagementConfig{}, "agent_management", ""))

	return diags
}
