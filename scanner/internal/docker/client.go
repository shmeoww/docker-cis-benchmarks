package docker

import "github.com/moby/moby/client"

// NewClient создаёт и возвращает клиент для общения с Docker.
// Возвращает либо готовый клиент, либо ошибку (вторым значением).
func NewClient() (*client.Client, error) {
	return client.New(client.FromEnv)
}