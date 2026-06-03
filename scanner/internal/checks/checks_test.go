package checks

import (
	"testing"

	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

// ─── Вспомогательные функции ─────────────────────────────────────────────────

// getImageCheck ищет проверку образа по ID; останавливает тест если не найдена.
func getImageCheck(t *testing.T, id string) ImageCheck {
	t.Helper()
	for _, c := range ImageChecks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("ImageCheck с ID=%q не найдена", id)
	return ImageCheck{}
}

// getContainerCheck ищет проверку контейнера по ID.
func getContainerCheck(t *testing.T, id string) ContainerCheck {
	t.Helper()
	for _, c := range ContainerChecks {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("ContainerCheck с ID=%q не найдена", id)
	return ContainerCheck{}
}

// assertStatus — вспомогательный assert с подробным выводом при ошибке.
func assertStatus(t *testing.T, result model.CheckResult, want model.Status) {
	t.Helper()
	if result.Status != want {
		t.Errorf("статус: хотели %s, получили %s\n  details: %q",
			want, result.Status, result.Details)
	}
}

// ─── Тесты проверок образов ───────────────────────────────────────────────────

func TestImageNoRootUser(t *testing.T) {
	c := getImageCheck(t, "image_no_root_user")
	tests := []struct {
		name     string
		user     string
		expected model.Status
	}{
		{"пустой USER → root", "", model.StatusFail},
		{"USER=root", "root", model.StatusFail},
		{"USER=0 (uid root)", "0", model.StatusFail},
		{"USER задан как app", "app", model.StatusPass},
		{"USER задан как uid 1000", "1000", model.StatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ImageData{User: tt.user})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestImageHasHealthcheck(t *testing.T) {
	c := getImageCheck(t, "image_has_healthcheck")
	tests := []struct {
		name           string
		hasHealthcheck bool
		expected       model.Status
	}{
		{"HEALTHCHECK не задан", false, model.StatusFail},
		{"HEALTHCHECK задан", true, model.StatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ImageData{HasHealthcheck: tt.hasHealthcheck})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestImageUseCopyNotAdd(t *testing.T) {
	c := getImageCheck(t, "image_use_copy_not_add")
	tests := []struct {
		name     string
		history  []string
		expected model.Status
	}{
		{"история пуста", nil, model.StatusPass},
		{"только COPY", []string{"/bin/sh -c #(nop)  COPY file:abc in /"}, model.StatusPass},
		{"есть ADD", []string{"/bin/sh -c #(nop)  ADD file:abc in /"}, model.StatusFail},
		{"ADD и COPY вместе", []string{
			"/bin/sh -c #(nop)  COPY file:abc in /",
			"/bin/sh -c #(nop)  ADD file:xyz in /tmp",
		}, model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ImageData{History: tt.history})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestImageNoLatestTag(t *testing.T) {
	c := getImageCheck(t, "image_no_latest_tag")
	tests := []struct {
		name     string
		tags     []string
		expected model.Status
	}{
		{"нет тегов (sha256)", nil, model.StatusFail},
		{"тег :latest", []string{"nginx:latest"}, model.StatusFail},
		{"конкретная версия", []string{"nginx:1.27.0"}, model.StatusPass},
		{"несколько тегов, latest есть", []string{"nginx:1.27", "nginx:latest"}, model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ImageData{Tags: tt.tags})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestImageNoSecretsInEnv(t *testing.T) {
	c := getImageCheck(t, "image_no_secrets_in_env")
	tests := []struct {
		name     string
		env      []string
		expected model.Status
	}{
		{"нет ENV", nil, model.StatusPass},
		{"безопасные переменные", []string{"PATH=/usr/bin", "MYSQL_VERSION=8.0"}, model.StatusPass},
		{"PASSWORD в имени", []string{"MYSQL_PASSWORD=secret"}, model.StatusFail},
		{"API_KEY", []string{"API_KEY=abc123"}, model.StatusFail},
		{"SECRET в имени", []string{"APP_SECRET=xyz"}, model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ImageData{Env: tt.env})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestImageNoPrivilegedPorts(t *testing.T) {
	c := getImageCheck(t, "image_no_privileged_ports")
	tests := []struct {
		name     string
		ports    []string
		expected model.Status
	}{
		{"нет портов", nil, model.StatusPass},
		{"порт 3306 (MySQL) — нормальный", []string{"3306/tcp"}, model.StatusPass},
		{"SSH-порт 22", []string{"22/tcp"}, model.StatusFail},
		{"привилегированный порт 80", []string{"80/tcp"}, model.StatusWarn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ImageData{ExposedPorts: tt.ports})
			assertStatus(t, result, tt.expected)
		})
	}
}

// ─── Тесты проверок контейнеров ───────────────────────────────────────────────

func TestContainerNoPrivileged(t *testing.T) {
	c := getContainerCheck(t, "container_no_privileged")
	tests := []struct {
		name       string
		privileged bool
		expected   model.Status
	}{
		{"не привилегированный", false, model.StatusPass},
		{"привилегированный", true, model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{Privileged: tt.privileged})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestContainerNoDockerSocket(t *testing.T) {
	c := getContainerCheck(t, "container_no_docker_socket")
	tests := []struct {
		name     string
		binds    []string
		expected model.Status
	}{
		{"нет монтирований", nil, model.StatusPass},
		{"обычный том", []string{"/data:/data"}, model.StatusPass},
		{"docker.sock примонтирован", []string{"/var/run/docker.sock:/var/run/docker.sock"}, model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{Binds: tt.binds})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestContainerNoNewPrivileges(t *testing.T) {
	c := getContainerCheck(t, "container_no_new_privileges")
	tests := []struct {
		name        string
		securityOpt []string
		expected    model.Status
	}{
		{"SecurityOpt не задан", nil, model.StatusFail},
		{"только apparmor", []string{"apparmor=docker-default"}, model.StatusFail},
		{"no-new-privileges задан (двоеточие)", []string{"no-new-privileges:true"}, model.StatusPass},
		{"no-new-privileges задан (равно)", []string{"no-new-privileges=true"}, model.StatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{SecurityOpt: tt.securityOpt})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestContainerNoHostNetwork(t *testing.T) {
	c := getContainerCheck(t, "container_no_host_network")
	tests := []struct {
		name        string
		networkMode string
		expected    model.Status
	}{
		{"bridge (обычная сеть)", "bridge", model.StatusPass},
		{"custom network", "my_network", model.StatusPass},
		{"host network", "host", model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{NetworkMode: tt.networkMode})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestContainerNoSensitiveMounts(t *testing.T) {
	c := getContainerCheck(t, "container_no_sensitive_mounts")
	tests := []struct {
		name     string
		binds    []string
		expected model.Status
	}{
		{"нет монтирований", nil, model.StatusPass},
		{"обычный том /data", []string{"/data:/data"}, model.StatusPass},
		{"корень /", []string{"/:/host"}, model.StatusFail},
		{"/etc примонтирован", []string{"/etc:/host-etc"}, model.StatusFail},
		{"/boot примонтирован", []string{"/boot:/host-boot"}, model.StatusFail},
		{"/etc/подкаталог", []string{"/etc/nginx:/config"}, model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{Binds: tt.binds})
			assertStatus(t, result, tt.expected)
		})
	}
}

func TestContainerSeccompAppArmor(t *testing.T) {
	c := getContainerCheck(t, "container_seccomp_apparmor")
	tests := []struct {
		name        string
		securityOpt []string
		expected    model.Status
	}{
		{"нет SecurityOpt — дефолтный профиль", nil, model.StatusPass},
		{"no-new-privileges (seccomp не тронут)", []string{"no-new-privileges:true"}, model.StatusPass},
		{"seccomp=unconfined", []string{"seccomp=unconfined"}, model.StatusFail},
		{"apparmor=unconfined", []string{"apparmor=unconfined"}, model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{SecurityOpt: tt.securityOpt})
			assertStatus(t, result, tt.expected)
		})
	}
}

// Тесты для 8 оставшихся проверок контейнера.
 
func TestContainerRestrictedCapabilities(t *testing.T) {
	c := getContainerCheck(t, "container_restricted_capabilities")
	tests := []struct {
		name     string
		caps     []string
		expected model.Status
	}{
		{"нет capabilities", nil, model.StatusPass},
		{"опасная SYS_ADMIN", []string{"SYS_ADMIN"}, model.StatusFail},
		{"опасная NET_ADMIN", []string{"NET_ADMIN"}, model.StatusFail},
		{"неопасная NET_BIND_SERVICE", []string{"NET_BIND_SERVICE"}, model.StatusWarn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{AddedCaps: tt.caps})
			assertStatus(t, result, tt.expected)
		})
	}
}
 
func TestContainerNoHostPid(t *testing.T) {
	c := getContainerCheck(t, "container_no_host_pid")
	tests := []struct {
		name     string
		pidMode  string
		expected model.Status
	}{
		{"PID изолирован (пусто)", "", model.StatusPass},
		{"PID хоста", "host", model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{PidMode: tt.pidMode})
			assertStatus(t, result, tt.expected)
		})
	}
}
 
func TestContainerNoHostIpc(t *testing.T) {
	c := getContainerCheck(t, "container_no_host_ipc")
	tests := []struct {
		name     string
		ipcMode  string
		expected model.Status
	}{
		{"IPC private", "private", model.StatusPass},
		{"IPC хоста", "host", model.StatusFail},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{IpcMode: tt.ipcMode})
			assertStatus(t, result, tt.expected)
		})
	}
}
 
func TestContainerReadonlyRootfs(t *testing.T) {
	c := getContainerCheck(t, "container_readonly_rootfs")
	tests := []struct {
		name     string
		readonly bool
		expected model.Status
	}{
		{"ФС для записи (предупреждение)", false, model.StatusWarn},
		{"ФС только для чтения", true, model.StatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{ReadonlyRootfs: tt.readonly})
			assertStatus(t, result, tt.expected)
		})
	}
}
 
func TestContainerMemoryLimit(t *testing.T) {
	c := getContainerCheck(t, "container_memory_limit")
	tests := []struct {
		name     string
		memory   int64
		expected model.Status
	}{
		{"лимит не задан (0)", 0, model.StatusWarn},
		{"лимит задан (512 МБ)", 512 * 1024 * 1024, model.StatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{MemoryLimit: tt.memory})
			assertStatus(t, result, tt.expected)
		})
	}
}
 
func TestContainerPidsLimit(t *testing.T) {
	c := getContainerCheck(t, "container_pids_limit")
	tests := []struct {
		name     string
		pids     int64
		expected model.Status
	}{
		{"лимит не задан", 0, model.StatusWarn},
		{"лимит задан (100)", 100, model.StatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{PidsLimit: tt.pids})
			assertStatus(t, result, tt.expected)
		})
	}
}
 
func TestContainerCPULimit(t *testing.T) {
	c := getContainerCheck(t, "container_cpu_limit")
	tests := []struct {
		name     string
		nanoCPUs int64
		expected model.Status
	}{
		{"лимит не задан", 0, model.StatusWarn},
		{"1 ядро (1e9 nanoCPU)", 1_000_000_000, model.StatusPass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{NanoCPUs: tt.nanoCPUs})
			assertStatus(t, result, tt.expected)
		})
	}
}
 
func TestContainerRestartPolicy(t *testing.T) {
	c := getContainerCheck(t, "container_restart_policy")
	tests := []struct {
		name     string
		policy   string
		expected model.Status
	}{
		{"no (нет перезапуска)", "no", model.StatusPass},
		{"on-failure (безопасная)", "on-failure", model.StatusPass},
		{"always (предупреждение)", "always", model.StatusWarn},
		{"unless-stopped (предупреждение)", "unless-stopped", model.StatusWarn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Run(docker.ContainerData{RestartPolicy: tt.policy})
			assertStatus(t, result, tt.expected)
		})
	}
}