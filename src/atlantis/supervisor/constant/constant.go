package constant

var (
	Region = "dev"
	Zone   = "dev"
)

const (
	SupervisorRPCVersion            = "1.0.0"
	DefaultSupervisorRPCPort        = uint16(1337)
	DefaultSupervisorSaveDir        = "/etc/atlantis/supervisor/save"
	DefaultSupervisorNumContainers  = uint16(100)
	DefaultSupervisorNumSecondary   = uint16(5)
	DefaultSupervisorMinPort        = uint16(61000)
	DefaultSupervisorCPUShares      = uint(100)
	DefaultSupervisorMemoryLimit    = uint(4096)
	DefaultSupervisorRegistryHost   = "localhost"
	DefaultResultDuration           = "30m"
	DefaultMaintenanceFile          = "/etc/atlantis/supervisor/maint"
	DefaultMaintenanceCheckInterval = "5s"
)
