package checks

import (
	"strconv"
	"strings"

	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

// ImageChecks — список всех CIS-проверок для Docker-образов
var ImageChecks = []ImageCheck{

	// Проверка 1 (CIS 4.1): образ не запускается от root
	{
		ID:           "image_no_root_user",
		Title:        "Образ не должен запускаться от root",
		CISReference: "4.1",
		Severity:     model.SeverityCritical,
		Eval: func(data docker.ImageData) (model.Status, string, string) {
			u := strings.TrimSpace(data.User)
			if u == "" || u == "0" || strings.ToLower(u) == "root" {
				return model.StatusFail,
					"USER не задан или равен root (значение: \"" + u + "\")",
					"Добавьте в Dockerfile инструкцию USER <непривилегированный-пользователь>."
			}
			return model.StatusPass, "USER задан: " + data.User, ""
		},
	},

	// Проверка 2 (CIS 4.6): HEALTHCHECK задан
	{
		ID:           "image_has_healthcheck",
		Title:        "В образе должен быть задан HEALTHCHECK",
		CISReference: "4.6",
		Severity:     model.SeverityMedium,
		Eval: func(data docker.ImageData) (model.Status, string, string) {
			if !data.HasHealthcheck {
				return model.StatusFail,
					"HEALTHCHECK не задан",
					"Добавьте в Dockerfile инструкцию HEALTHCHECK для мониторинга состояния сервиса."
			}
			return model.StatusPass, "HEALTHCHECK задан", ""
		},
	},

	// Проверка 3 (CIS 4.9): COPY вместо ADD
	{
		ID:           "image_use_copy_not_add",
		Title:        "Использовать COPY вместо ADD",
		CISReference: "4.9",
		Severity:     model.SeverityLow,
		Eval: func(data docker.ImageData) (model.Status, string, string) {
			// В истории образа инструкции Dockerfile помечены #(nop).
			// RUN-команды #(nop) не содержат, поэтому ложных срабатываний нет.
			for _, layer := range data.History {
				if strings.Contains(layer, "#(nop)") &&
					strings.Contains(strings.ToUpper(layer), " ADD ") {
					return model.StatusFail,
						"Обнаружена инструкция ADD в истории образа",
						"Замените ADD на COPY. ADD допустим только для распаковки локальных архивов."
				}
			}
			return model.StatusPass, "Инструкция ADD не обнаружена", ""
		},
	},

	// Проверка 4 (best-practice): версия образа зафиксирована
	{
		ID:       "image_no_latest_tag",
		Title:    "Версия образа должна быть зафиксирована (не :latest)",
		Severity: model.SeverityMedium,
		Eval: func(data docker.ImageData) (model.Status, string, string) {
			if len(data.Tags) == 0 {
				return model.StatusFail,
					"Образ без тега — версия не зафиксирована",
					"Указывайте конкретную версию в Dockerfile: FROM образ:1.2.3"
			}
			for _, tag := range data.Tags {
				if strings.HasSuffix(tag, ":latest") {
					return model.StatusFail,
						"Используется тег :latest: " + tag,
						"Замените :latest на конкретную версию."
				}
			}
			return model.StatusPass, "Версия зафиксирована: " + data.Tags[0], ""
		},
	},

	// Проверка 5 (CIS 4.10): секреты не в переменных окружения
	{
		ID:           "image_no_secrets_in_env",
		Title:        "Переменные окружения не должны содержать секреты",
		CISReference: "4.10",
		Severity:     model.SeverityHigh,
		Eval: func(data docker.ImageData) (model.Status, string, string) {
			suspects := []string{"PASSWORD", "SECRET", "TOKEN", "PASSWD", "PRIVATE", "CREDENTIALS", "API_KEY"}
			for _, env := range data.Env {
				parts := strings.SplitN(env, "=", 2)
				key := strings.ToUpper(parts[0])
				for _, s := range suspects {
					if strings.Contains(key, s) {
						return model.StatusFail,
							"Подозрительная переменная ENV: " + parts[0],
							"Не храните секреты в ENV. Используйте Docker secrets или переменные при запуске."
					}
				}
			}
			return model.StatusPass, "Подозрительных переменных ENV не найдено", ""
		},
	},

	// Проверка 6 (CIS 5.8): нет SSH-порта и привилегированных портов в EXPOSE
	{
		ID:           "image_no_privileged_ports",
		Title:        "Образ не должен открывать SSH-порт или привилегированные порты",
		CISReference: "5.8",
		Severity:     model.SeverityHigh,
		Eval: func(data docker.ImageData) (model.Status, string, string) {
			for _, port := range data.ExposedPorts {
				parts := strings.SplitN(port, "/", 2)
				num, err := strconv.Atoi(parts[0])
				if err != nil {
					continue
				}
				if num == 22 {
					return model.StatusFail,
						"Открыт SSH-порт (22): " + port,
						"Не запускайте SSH внутри контейнера. Управляйте контейнером через Docker CLI."
				}
				if num < 1024 {
					return model.StatusWarn,
						"Открыт привилегированный порт: " + port,
						"По возможности используйте порты >= 1024."
				}
			}
			return model.StatusPass, "Привилегированных и SSH-портов не открыто", ""
		},
	},
}