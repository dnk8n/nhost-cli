package compose

const (
	// default ports
	serverDefaultPort                  = 1337
	svcPostgresDefaultPort             = 5432
	svcHasuraDefaultPort               = 8080
	svcMailhogDefaultPort              = 1025
	hasuraConsoleMigrateAPIDefaultPort = 9693
)

type Ports map[string]uint32

func DefaultPorts() Ports {
	return Ports{
		SvcTraefik:       serverDefaultPort,
		SvcPostgres:      svcPostgresDefaultPort,
		SvcGraphqlEngine: svcHasuraDefaultPort,
		SvcMailhog:       svcMailhogDefaultPort,
		SvcHasuraConsole: hasuraConsoleMigrateAPIDefaultPort,
	}
}

func NewPorts(proxy uint32) Ports {
	p := DefaultPorts()
	p[SvcTraefik] = proxy
	return p
}
