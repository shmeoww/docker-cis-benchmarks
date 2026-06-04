package checks

import (
	"context"
	"sync"
	"time"

	"github.com/moby/moby/client"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/docker"
	"github.com/shmeoww/docker-cis-benchmarks/scanner/internal/model"
)

// Version — версия сканера, попадает в каждый ScanReport
const Version = "0.1.0"

// ScanImage собирает данные образа и прогоняет все проверки параллельно
func ScanImage(ctx context.Context, cli *client.Client, ref string) (model.ScanReport, error) {
	data, err := docker.CollectImage(ctx, cli, ref)
	if err != nil {
		return model.ScanReport{}, err
	}
	results := runImageChecks(data)
	return model.ScanReport{
		ScannerVersion: Version,
		ScannedAt:      time.Now(),
		Target:         model.Target{Type: "image", ID: data.ID, Name: ref},
		Summary:        model.ComputeSummary(results),
		Checks:         results,
	}, nil
}

// ScanContainer собирает данные контейнера и прогоняет все проверки параллельно
func ScanContainer(ctx context.Context, cli *client.Client, ref string) (model.ScanReport, error) {
	data, err := docker.CollectContainer(ctx, cli, ref)
	if err != nil {
		return model.ScanReport{}, err
	}
	results := runContainerChecks(data)
	return model.ScanReport{
		ScannerVersion: Version,
		ScannedAt:      time.Now(),
		Target:         model.Target{Type: "container", ID: data.ID, Name: data.Name},
		Summary:        model.ComputeSummary(results),
		Checks:         results,
	}, nil
}

// ScanAll сканирует все локальные образы и все контейнеры
// Возвращает срез отчётов — по одному на каждую цель
func ScanAll(ctx context.Context, cli *client.Client) ([]model.ScanReport, error) {
	var reports []model.ScanReport

	// Образы
	images, err := cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	for _, img := range images.Items {
		ref := img.ID
		if len(img.RepoTags) > 0 {
			ref = img.RepoTags[0]
		}
		report, err := ScanImage(ctx, cli, ref)
		if err != nil {
			continue // пропускаем недоступные образы
		}
		reports = append(reports, report)
	}

	// Контейнеры
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	for _, c := range containers.Items {
		report, err := ScanContainer(ctx, cli, c.ID)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// runImageChecks запускает каждую проверку в отдельной горутине
// Результаты собираются через буферизованный канал
func runImageChecks(data docker.ImageData) []model.CheckResult {
	// Буфер = число проверок, горутины не блокируются на отправке
	ch := make(chan model.CheckResult, len(ImageChecks))
	var wg sync.WaitGroup

	for _, c := range ImageChecks {
		wg.Add(1)
		// Передаём check как аргумент — каждая горутина получает свою копию
		go func(check ImageCheck) {
			defer wg.Done()
			ch <- check.Run(data)
		}(c)
	}

	wg.Wait()  // ждём завершения всех горутин
	close(ch)  // закрываем канал — сигнал, что данных больше не будет

	var results []model.CheckResult
	for r := range ch { // читаем все результаты из канала
		results = append(results, r)
	}
	return results
}

// runContainerChecks — то же самое для проверок контейнера
func runContainerChecks(data docker.ContainerData) []model.CheckResult {
	ch := make(chan model.CheckResult, len(ContainerChecks))
	var wg sync.WaitGroup

	for _, c := range ContainerChecks {
		wg.Add(1)
		go func(check ContainerCheck) {
			defer wg.Done()
			ch <- check.Run(data)
		}(c)
	}

	wg.Wait()
	close(ch)

	var results []model.CheckResult
	for r := range ch {
		results = append(results, r)
	}
	return results
}