package docker

import (
	"context"

	"github.com/moby/moby/client"
)

// ContainerData — нормализованные данные одного контейнера для CIS-проверок запущенных контейнеров
type ContainerData struct {
	ID             string
	Name           string
	Image          string
	Privileged     bool     // --privileged
	AddedCaps      []string // добавленные capabilities (CapAdd)
	NetworkMode    string   // "host" = сеть хоста
	PidMode        string   // "host" = PID-namespace хоста
	IpcMode        string   // "host" = IPC-namespace хоста
	Binds          []string // монтирования "источник:цель"
	SecurityOpt    []string // seccomp/apparmor/no-new-privileges
	ReadonlyRootfs bool     // --read-only
	MemoryLimit    int64    // лимит памяти в байтах (0 = не задан)
	NanoCPUs       int64    // лимит CPU (0 = не задан)
	PidsLimit      int64    // лимит процессов (0 = не задан)
	RestartPolicy  string   // имя политики перезапуска
	MaxRetry       int      // макс. число перезапусков
}

// CollectContainer инспектирует один контейнер (по ID или имени) и возвращает заполненную структуру ContainerData.
func CollectContainer(ctx context.Context, cli *client.Client, ref string) (ContainerData, error) {
	result, err := cli.ContainerInspect(ctx, ref, client.ContainerInspectOptions{})
	if err != nil {
		return ContainerData{}, err
	}

	// Данные контейнера лежат в поле result.Container
	c := result.Container

	data := ContainerData{
		ID:    c.ID,
		Name:  c.Name,
		Image: c.Image,
	}

	// HostConfig — указатель, может быть nil, проверяем
	hc := c.HostConfig
	if hc != nil {
		data.Privileged = hc.Privileged
		data.AddedCaps = []string(hc.CapAdd)
		data.NetworkMode = string(hc.NetworkMode)
		data.PidMode = string(hc.PidMode)
		data.IpcMode = string(hc.IpcMode)
		data.Binds = hc.Binds
		data.SecurityOpt = hc.SecurityOpt
		data.ReadonlyRootfs = hc.ReadonlyRootfs
		data.MemoryLimit = hc.Memory
		data.NanoCPUs = hc.NanoCPUs
		if hc.PidsLimit != nil {
			data.PidsLimit = *hc.PidsLimit
		}
		data.RestartPolicy = string(hc.RestartPolicy.Name)
		data.MaxRetry = hc.RestartPolicy.MaximumRetryCount
	}

	return data, nil
}