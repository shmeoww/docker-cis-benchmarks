package docker

import (
	"context"
	
	"github.com/moby/moby/client"
)

// ImageData — нормализованные данные одного образа, только то, что нужно CIS-проверкам.
type ImageData struct {
	ID             string   // полный ID образа
	Tags           []string // имена/теги, например ["mysql:8.0"]
	User           string   // инструкция USER (пустая строка = запуск от root)
	HasHealthcheck bool     // задан ли HEALTHCHECK
	ExposedPorts   []string // открытые порты, напр. ["3306/tcp"]
	Env            []string // переменные окружения вида "KEY=value"
	History        []string // строки инструкций сборки (для поиска ADD/COPY)
}

// CollectImage инспектирует один образ по ссылке (имя:тег или ID) и возвращает заполненную структуру ImageData.
func CollectImage(ctx context.Context, cli *client.Client, ref string) (ImageData, error) {
	// Запрашиваем подробную информацию об образе.
	inspect, err := cli.ImageInspect(ctx, ref)
	if err != nil {
		return ImageData{}, err
	}

	data := ImageData{
		ID:   inspect.ID,
		Tags: inspect.RepoTags,
	}

	// Config — указатель, может быть nil; проверяем перед обращением.
	if inspect.Config != nil {
		data.User = inspect.Config.User
		data.HasHealthcheck = inspect.Config.Healthcheck != nil
		data.Env = inspect.Config.Env

		// ExposedPorts — это map (ключ = порт). Берём ключи.
		for port := range inspect.Config.ExposedPorts {
			data.ExposedPorts = append(data.ExposedPorts, port)
		}
	}

	// История слоёв — нужна для проверки "ADD вместо COPY".
	history, err := cli.ImageHistory(ctx, ref)
	if err != nil {
		return ImageData{}, err
	}
	for _, layer := range history.Items {
		data.History = append(data.History, layer.CreatedBy)
	}

	return data, nil
}