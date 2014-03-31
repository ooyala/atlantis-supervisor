/* Copyright 2014 Ooyala, Inc. All rights reserved.
 *
 * This file is licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License is
 * distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and limitations under the License.
 */

package constant

var (
	Region = "dev"
	Zone   = "dev"
)

const (
	SupervisorRPCVersion            = "3.0.0"
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
	ContainerLogDir                 = "/var/log/atlantis"
)
