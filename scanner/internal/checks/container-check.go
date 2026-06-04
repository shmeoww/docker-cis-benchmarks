package checks

import (
	"fmt"
	"strings"

	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

// ContainerChecks — список всех CIS-проверок для запущенных контейнеров
var ContainerChecks = []ContainerCheck{

	// Проверка 7 (CIS 5.4): не привилегированный режим
	{
		ID:           "container_no_privileged",
		Title:        "Контейнер не должен запускаться в привилегированном режиме",
		CISReference: "5.4",
		Severity:     model.SeverityCritical,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			if data.Privileged {
				return model.StatusFail,
					"Контейнер запущен с флагом --privileged",
					"Уберите --privileged. Если нужны отдельные возможности — используйте --cap-add."
			}
			return model.StatusPass, "Привилегированный режим не используется", ""
		},
	},

	// Проверка 8 (CIS 5.3): capabilities ограничены
	{
		ID:           "container_restricted_capabilities",
		Title:        "Опасные Linux capabilities не должны быть добавлены",
		CISReference: "5.3",
		Severity:     model.SeverityHigh,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			dangerous := []string{"SYS_ADMIN", "NET_ADMIN", "SYS_PTRACE", "SYS_MODULE", "SYS_RAWIO", "DAC_OVERRIDE", "SETUID", "SETGID", "NET_RAW"}
			for _, cap := range data.AddedCaps {
				for _, d := range dangerous {
					if strings.EqualFold(cap, d) {
						return model.StatusFail,
							"Добавлена опасная capability: " + cap,
							"Уберите capability " + cap + " или замените на минимально необходимую."
					}
				}
			}
			if len(data.AddedCaps) > 0 {
				return model.StatusWarn,
					"Добавлены capabilities: " + strings.Join(data.AddedCaps, ", "),
					"Проверьте, действительно ли перечисленные capabilities необходимы."
			}
			return model.StatusPass, "Дополнительные capabilities не добавлены", ""
		},
	},

	// Проверка 9 (CIS 5.9): не используется сеть хоста
	{
		ID:           "container_no_host_network",
		Title:        "Контейнер не должен использовать сеть хоста",
		CISReference: "5.9",
		Severity:     model.SeverityHigh,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			if data.NetworkMode == "host" {
				return model.StatusFail,
					"Контейнер использует сетевой стек хоста (--net=host)",
					"Уберите --net=host. Используйте Docker-сети для межсервисного общения."
			}
			return model.StatusPass, "Сетевой режим: " + data.NetworkMode, ""
		},
	},

	// Проверка 10 (CIS 5.15): не разделяется PID-namespace хоста
	{
		ID:           "container_no_host_pid",
		Title:        "Контейнер не должен разделять PID-namespace хоста",
		CISReference: "5.15",
		Severity:     model.SeverityHigh,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			if data.PidMode == "host" {
				return model.StatusFail,
					"Контейнер разделяет PID-namespace хоста (--pid=host)",
					"Уберите --pid=host. Контейнер видит и может влиять на процессы хоста."
			}
			return model.StatusPass, "PID-namespace изолирован", ""
		},
	},

	// Проверка 11 (CIS 5.16): не разделяется IPC-namespace хоста
	{
		ID:           "container_no_host_ipc",
		Title:        "Контейнер не должен разделять IPC-namespace хоста",
		CISReference: "5.16",
		Severity:     model.SeverityMedium,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			if data.IpcMode == "host" {
				return model.StatusFail,
					"Контейнер разделяет IPC-namespace хоста (--ipc=host)",
					"Уберите --ipc=host. Это открывает доступ к разделяемой памяти хоста."
			}
			return model.StatusPass, "IPC-namespace изолирован: " + data.IpcMode, ""
		},
	},

	// Проверка 12 (CIS 5.31): docker.sock не примонтирован
	{
		ID:           "container_no_docker_socket",
		Title:        "Docker-сокет не должен быть примонтирован внутрь контейнера",
		CISReference: "5.31",
		Severity:     model.SeverityCritical,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			for _, bind := range data.Binds {
				if strings.Contains(bind, "docker.sock") {
					return model.StatusFail,
						"Docker-сокет примонтирован: " + bind,
						"Уберите монтирование docker.sock. Контейнер с доступом к нему может захватить весь Docker-хост."
				}
			}
			return model.StatusPass, "Docker-сокет не примонтирован", ""
		},
	},

	// Проверка 13 (CIS 5.25): флаг no-new-privileges установлен
	{
		ID:           "container_no_new_privileges",
		Title:        "Должен быть установлен флаг no-new-privileges",
		CISReference: "5.25",
		Severity:     model.SeverityHigh,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			for _, opt := range data.SecurityOpt {
				if strings.Contains(strings.ToLower(opt), "no-new-privileges") {
					return model.StatusPass, "no-new-privileges установлен: " + opt, ""
				}
			}
			return model.StatusFail,
				"Флаг no-new-privileges не установлен",
				"Добавьте --security-opt=no-new-privileges:true при запуске контейнера."
		},
	},

	// Проверка 14 (CIS 5.12): корневая ФС только для чтения
	{
		ID:           "container_readonly_rootfs",
		Title:        "Корневая ФС контейнера должна быть только для чтения",
		CISReference: "5.12",
		Severity:     model.SeverityMedium,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			if !data.ReadonlyRootfs {
				return model.StatusWarn,
					"Корневая ФС доступна для записи",
					"Запустите с флагом --read-only. Для изменяемых данных используйте тома (--volume)."
			}
			return model.StatusPass, "Корневая ФС смонтирована только для чтения", ""
		},
	},

	// Проверка 15 (CIS 5.10): лимит памяти задан
	{
		ID:           "container_memory_limit",
		Title:        "Должен быть задан лимит памяти",
		CISReference: "5.10",
		Severity:     model.SeverityMedium,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			if data.MemoryLimit <= 0 {
				return model.StatusWarn,
					"Лимит памяти не задан — контейнер может исчерпать ресурсы хоста",
					"Задайте лимит: docker run --memory=512m ..."
			}
			return model.StatusPass,
				fmt.Sprintf("Лимит памяти: %d МБ", data.MemoryLimit/1024/1024), ""
		},
	},

	// Проверка 16 (CIS 5.28): лимит PIDs задан
	{
		ID:           "container_pids_limit",
		Title:        "Должен быть задан лимит числа процессов (PIDs)",
		CISReference: "5.28",
		Severity:     model.SeverityLow,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			if data.PidsLimit <= 0 {
				return model.StatusWarn,
					"Лимит PIDs не задан",
					"Задайте лимит: docker run --pids-limit=100 ..."
			}
			return model.StatusPass, fmt.Sprintf("Лимит PIDs: %d", data.PidsLimit), ""
		},
	},

	// Проверка 17 (CIS 5.11): лимит CPU задан
	{
		ID:           "container_cpu_limit",
		Title:        "Должен быть задан лимит CPU",
		CISReference: "5.11",
		Severity:     model.SeverityMedium,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			if data.NanoCPUs <= 0 {
				return model.StatusWarn,
					"Лимит CPU не задан",
					"Задайте лимит: docker run --cpus=1.0 ..."
			}
			return model.StatusPass,
				fmt.Sprintf("Лимит CPU: %.2f ядер", float64(data.NanoCPUs)/1e9), ""
		},
	},

	// Проверка 18 (CIS 5.14): политика перезапуска безопасна
	{
		ID:           "container_restart_policy",
		Title:        "Политика перезапуска должна быть безопасной",
		CISReference: "5.14",
		Severity:     model.SeverityLow,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			switch data.RestartPolicy {
			case "always":
				return model.StatusWarn,
					"Политика перезапуска: always (бесконечные перезапуски)",
					"Используйте 'on-failure' с ограничением: --restart=on-failure:5"
			case "unless-stopped":
				return model.StatusWarn,
					"Политика перезапуска: unless-stopped",
					"Рассмотрите использование 'on-failure' с лимитом попыток."
			}
			return model.StatusPass, "Политика перезапуска: " + data.RestartPolicy, ""
		},
	},

	// Проверка 19 (CIS 5.5): чувствительные каталоги хоста не примонтированы
	{
		ID:           "container_no_sensitive_mounts",
		Title:        "Чувствительные каталоги хоста не должны быть примонтированы",
		CISReference: "5.5",
		Severity:     model.SeverityCritical,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			sensitive := []string{"/etc", "/boot", "/proc", "/sys", "/var/run"}
			for _, bind := range data.Binds {
				// Берём только источник (часть до первого ":").
				source := strings.SplitN(bind, ":", 2)[0]
				if source == "/" {
					return model.StatusFail,
						"Примонтирован корневой каталог хоста: " + bind,
						"Немедленно уберите монтирование / — это полный доступ к хосту."
				}
				for _, dir := range sensitive {
					if source == dir || strings.HasPrefix(source, dir+"/") {
						return model.StatusFail,
							"Примонтирован чувствительный каталог: " + bind,
							"Уберите монтирование " + source + "."
					}
				}
			}
			return model.StatusPass, "Чувствительные каталоги хоста не примонтированы", ""
		},
	},

	// Проверка 20 (CIS 5.1 / 5.21): seccomp и AppArmor не отключены
	{
		ID:           "container_seccomp_apparmor",
		Title:        "seccomp и AppArmor не должны быть явно отключены",
		CISReference: "5.21",
		Severity:     model.SeverityHigh,
		Eval: func(data docker.ContainerData) (model.Status, string, string) {
			for _, opt := range data.SecurityOpt {
				lower := strings.ToLower(opt)
				if strings.Contains(lower, "seccomp") && strings.Contains(lower, "unconfined") {
					return model.StatusFail,
						"seccomp явно отключён: " + opt,
						"Уберите --security-opt seccomp=unconfined. Используйте профиль по умолчанию."
				}
				if strings.Contains(lower, "apparmor") && strings.Contains(lower, "unconfined") {
					return model.StatusFail,
						"AppArmor явно отключён: " + opt,
						"Уберите --security-opt apparmor=unconfined."
				}
			}
			return model.StatusPass, "seccomp и AppArmor не отключены явно", ""
		},
	},
}